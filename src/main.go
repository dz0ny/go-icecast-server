package main

import (
	"bufio"
	"bytes"
	"decoder/ogg"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"utils"
)

type Audiocast struct {
	Artist      string `json:"artist"`
	Song        string `json:"song"`
	Encoder     string `json:"encoder"`
	Name        string `json:"station-name"`
	Description string `json:"station-description"`
	Audio       string `json:"station-info"`
	Type        string `json:"content-type"`
}

type Client struct {
	req          http.Request
	conn         net.Conn
	sent_packets uint32
	stream_end   chan int
}

//max 16 clients
var povezave = make(map[string]Audiocast, 16)
var klienti = make(map[net.Addr]Client, 128)

//icecast2 update
func parseMetadataUpdate(conn net.Conn, req *http.Request) {
	//example GET /admin/metadata?mode=updinfo&mount=/mixx&song=Test%20%2d%20Test
	mode := req.URL.Query().Get("mode")
	mount := req.URL.Query().Get("mount")
	song := req.URL.Query().Get("song")
	if *_DEBUGME {
		log.Println("mode", mode)
		log.Println("mount", mount)
		log.Println("song", song)
	}
	if povezava, ok := povezave[mount]; mode == "updinfo" && ok && len(song) > 0 && strings.Contains(song, "-") {
		s := strings.Split(song, " - ")
		conn.Write([]byte("HTTP/1.0 200 OK\r\n\r\nUpdated"))
		povezava.Artist = s[0]
		povezava.Song = s[1]
		povezave[mount] = povezava
	} else {
		conn.Write([]byte("HTTP/1.0 404 Not Found\r\n\r\n"))
	}
}

//icecast1 update
func parseOGG(conn net.Conn, povezava *Audiocast) {

	var oggPbefore bytes.Buffer

	f, _ := os.OpenFile("test.ogg", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)

	conn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))

	for {
		// CALLING FOR REWRITE WITH CHANNELS
		var data [1024 * 16]byte

		skip := oggPbefore.Len()
		if skip > 0 {
			oggPbefore.Read(data[0:])
			oggPbefore.Reset()
		}

		read, err := conn.Read(data[skip:])
		if err != nil {
			break
		}
		packet, _, next := ogg.NewOggpacket(data[0:read+skip], 0)
		if packet.Info != nil {
			(*povezava).Artist = packet.Info.Artist
			(*povezava).Song = packet.Info.Song
			(*povezava).Encoder = packet.Info.Encoder
		}

		if next > 0 {
			//pusti za naslednji paket
			f.Write(data[0:next])
			writeOggStreamToClients(data[0:next])
			//shrani se ne prepozane pakete
			oggPbefore.Write(data[next : read+skip])
		} else {
			//zapisivse
			f.Write(data[0 : read+skip])
			writeOggStreamToClients(data[0 : read+skip])
		}

	}
	return
}

func bos(data *[]byte) {

	//fix header_type_flag 5-6
	(*data)[5] = 2
	fix_packet(data, 0)

}

func fix_packet(data *[]byte, pn uint32) {

	//fix page_sequence_no 18-22
	var page_sequence_no = make([]byte, 4)
	binary.LittleEndian.PutUint32(page_sequence_no, pn)
	copy((*data)[18:23], page_sequence_no)

}

func writeOggStreamToClients(data []byte) {
	for pov, klient := range klienti {
		if klient.sent_packets == 0 {
			bos(&data)
		} else {
			fix_packet(&data, klient.sent_packets)
		}

		klient.conn.Write(data[0:])
		klient.sent_packets++
		klienti[pov] = klient
	}
}

func control_server_handle(conn net.Conn, basic_auth string) {
	povezava := new(Audiocast)

	if *_DEBUGME {
		log.Println("client", conn.RemoteAddr(), "connected")
	}

	for {
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			conn.Write([]byte("HTTP/1.0 500 Error\r\n\r\n"))
			utils.CheckError(err)
			break
		}
		//chech for authorization
		auth := req.Header.Get("Authorization")
		if len(auth) != 0 && len(basic_auth) > 1 {
			if !strings.Contains(auth, "Basic "+basic_auth) {
				conn.Write([]byte("HTTP/1.0 402 Not Authorized\r\n\r\nDon't poke here!"))
				break
			}
		} else if len(basic_auth) > 1 {
			conn.Write([]byte("HTTP/1.0 401 Not Authorized\r\nWWW-Authenticate: Basic realm=\"Icecast Server\"r\n"))
			break
		}

		if req.Method == "SOURCE" {

			//check for streams limit
			if len(povezave) >= 16 {
				conn.Write([]byte("HTTP/1.0 405 Too many streams\r\n\r\nToo many streams"))
				break
			}

			(*povezava).Type = req.Header.Get("Content-Type")
			(*povezava).Name = req.Header.Get("Ice-Name")
			(*povezava).Description = req.Header.Get("Ice-Description")
			(*povezava).Audio = req.Header.Get("Ice-Audio-Info")

			povezave[req.URL.Path] = *povezava

			//icecast 1 update
			parseOGG(conn, povezava)

			//utils.Cleanup
			delete(povezave, req.URL.Path)
			break
		} else if req.Method == "GET" && req.URL.Path == "/admin/metadata" {
			parseMetadataUpdate(conn, req)
			break
		} else {
			conn.Write([]byte("HTTP/1.0 405 Method not allowed\r\n\r\nMethod not allowed"))
			break
		}

	}
	if *_DEBUGME {
		log.Println("client", conn.RemoteAddr(), "disconnected")
	}
	conn.Close()
}

func control_server(port string, basic_auth string) {

	fmt.Println("icecast/shoutcast server running on port ", port)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", port)
	utils.CheckError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	utils.CheckError(err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go control_server_handle(conn, basic_auth)

	}

}

func play(w http.ResponseWriter, req *http.Request) {
	povezava := new(Client)

	hj, ok := w.(http.Hijacker)

	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	conn, _, hijack_err := hj.Hijack()
	if hijack_err != nil {
		conn.Write([]byte("HTTP/1.0 500 Error\r\n\r\n"))
		conn.Close()
		return
	}

	povezava.req = *req
	povezava.conn = conn
	povezava.sent_packets = 0
	povezava.stream_end = make(chan int)
	klienti[conn.RemoteAddr()] = *povezava
	stream, stream_exists := povezave["/mixx"]
	log.Println("stream", stream)
	log.Println("stream_exists", stream_exists)

	if !stream_exists {
		conn.Write([]byte("HTTP/1.0 4040 Not Found\r\n\r\nStream doesn't exist"))
		conn.Close()
		return
	}

	//introduce me
	conn.Write([]byte("HTTP/1.0 200 OK\r\nContent-Type:application/ogg\r\n\r\n"))

	//blocking channel
	<-povezava.stream_end
	log.Println("prekini", conn.RemoteAddr())
	conn.Close()
	delete(klienti, conn.RemoteAddr())
	return
}
func info(res http.ResponseWriter, req *http.Request) {

	res.Header().Set(
		"Content-Type",
		"application/json",
	)

	klienti, err := json.Marshal(povezave)
	if err != nil {
		fmt.Println("error:", err)
	}
	io.WriteString(res, utils.Stringify(klienti))
}

var server_port = flag.Int("i", 8000, "icecast server port")
var info_port = flag.Int("c", 3000, "web server port")
var user = flag.String("u", "", "icecast server username")
var password = flag.String("p", "", "icecast server password")
var _DEBUGME = flag.Bool("d", false, "enable debugging")

func main() {

	//icecast server
	go control_server(utils.ToIfPort(*server_port), utils.Basic_auth(*user, *password))

	//info server
	http.HandleFunc("/play/", play)
	http.HandleFunc("/info.json", info)
	http.Handle("/", http.FileServer(http.Dir("web/public")))

	go http.ListenAndServe(utils.ToIfPort(*info_port), nil)

	// infinite loop; don't use for, this is not c
	select {}

}
