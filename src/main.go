package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
)

type Icecast struct {
	path      string
	izvajalec string
	pesem     string
}

var povezave = make(map[string]Icecast)

func checkError(err error) {
	if err != nil {
		log.Fatal("Fatal error: ", err.Error())
	}
}

func clean(tag []byte) string {
	start := bytes.Index(tag, []byte("="))
	end := len(tag)
	for i := 0; i < end; i++ {
		if tag[i] < 32 {
			end = i
			break
		}
	}
	return stringify(tag[start+1 : end])
}

func stringify(tag []byte) string {
	data := bytes.NewBuffer(tag[0:])
	return data.String()
}

func toIfPort(port int) string {
	service := strconv.AppendInt([]byte(":"), int64(port), 10)
	return stringify(service)
}

func parseOGG(conn net.Conn, povezava *Icecast) {
	conn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	red := bufio.NewReader(conn)
	var vorbis [4096]byte
	for {
		n, err := red.Read(vorbis[0:])
		if err != nil {
			break
		}

		if n > 0 {
			//log.Print("prebral ", n/8)
			//vorbis data packet
			if bytes.Contains(vorbis[0:4], []byte("OggS")) {
				///79 103 103 83
				//OggS
				//log.Print("OggS serial = ", join(vorbis[12:16]))
				//log.Print("OggS page = ", vorbis[16:19])
				//log.Print("OggS segmet = ", vorbis[19:21])
				ARTIST := bytes.Index(vorbis[0:], []byte("ARTIST="))

				//log.Print("VORBIS > ", VORBIS)

				if ARTIST != -1 {
					TITLE := bytes.Index(vorbis[0:], []byte("TITLE="))
					povezava.izvajalec = clean(vorbis[ARTIST:TITLE])
					povezava.pesem = clean(vorbis[TITLE : TITLE+140])
					//log.Print("ARTIST ", povezava.izvajalec)
					//log.Print("TITLE ", povezava.pesem)
					povezave[conn.RemoteAddr().String()] = *povezava
				}
			}
		} else {
			break
		}

	}
}

func control_server_handle(conn net.Conn) {
	povezava := new(Icecast)
	povezave[conn.RemoteAddr().String()] = *povezava
	log.Println("client", conn.RemoteAddr(), "connected")
	for {
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			conn.Write([]byte("HTTP/1.0 500 Error\r\n\r\n"))
			return
		}

		if req.Method == "SOURCE" {
			povezava.path = req.URL.Path
			parseOGG(conn, povezava)
			break
		} else {
			conn.Write([]byte("HTTP/1.0 405 Method not allowed\r\n\r\nMethod not allowed"))
			break
		}

	}
	log.Println("client", conn.RemoteAddr(), "disconnected")
	delete(povezave, conn.RemoteAddr().String())
	conn.Close()
}

func control_server(port string) {

	fmt.Println("Icecast server running on port ", port)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", port)
	checkError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go control_server_handle(conn)

	}

}

func info(res http.ResponseWriter, req *http.Request) {
	res.Header().Set(
		"Content-Type",
		"text/plain",
	)
	klienti, err := json.Marshal(povezave)
	if err != nil {
		fmt.Println("error:", err)
	}
	io.WriteString(res, stringify(klienti))
}

func main() {

	//cli
	var server_port = flag.Int("port", 8000, "icecast server port")
	var info_port = flag.Int("web_port", 3000, "web server port")
	flag.Parse()

	//icecast server
	go control_server(toIfPort(*server_port))

	//info server
	http.HandleFunc("/", info)

	go http.ListenAndServe(toIfPort(*info_port), nil)

	// infinite loop; don't use form, this is not c
	select {}
}
