package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
)

type Audiocast struct {
	Artist string `json:"artist"`
	Song   string `json:"song"`
}

type OggPacket struct {
	Version          uint64
	Header_type      uint64
	Granule_position uint64
	Sequence         uint64
	Serial_number    uint64
	Crc              uint64
	Segments         uint64
}

//max 16 clients
var povezave = make(map[string]Audiocast, 16)
var _DEBUGME = false

func checkError(err error) {
	if err != nil {
		log.Println("Fatal error: ", err.Error())
	}
}

func clean(tag []byte) string {
	start := bytes.Index(tag, []byte("="))
	end := len(tag)
	for i := 0; i < end; i++ {
		if tag[i] < 32 {
			end = i
			//lookahead
			if tag[i+1] < 32 {
				break
			}
		}
	}
	if false {
		log.Println(tag)
		log.Println(tag[start+1 : end])
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

func varint(slice []byte) uint64 {
	number, _ := binary.Uvarint(slice)
	return number
}

func parseOGG(conn net.Conn, povezava *Audiocast, path *string) {
	conn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	red := bufio.NewReader(conn)
	var vorbis [4096]byte
	alsoReadNext := 0
	for {
		n, err := red.Read(vorbis[0:])
		if err != nil {
			checkError(err)
			break
		}

		if n > 0 {

			//vorbis data packet
			if bytes.Contains(vorbis[0:4], []byte("OggS")) {
				packet := new(OggPacket)
				//http://wiki.xiph.org/Ogg_Skeleton_4
				//79 103 103 83  | 0-3 header
				// 0 | 4-5 version
				// 4 | 5-6 type
				// 0 48 42 0 - 0 0 0 0 | 6-13 granule
				// 172 79 0 0 | 14-17 serial_number
				// 241 0 0 0 | 18-21 sequence
				// 196 103 18 176 | 22-25
				//38 93
				(*packet).Version = varint(vorbis[4:5])
				(*packet).Header_type = varint(vorbis[5:6])
				(*packet).Granule_position = varint(vorbis[6:14])
				(*packet).Serial_number = varint(vorbis[14:18])
				(*packet).Sequence = varint(vorbis[18:22])
				(*packet).Crc = varint(vorbis[22:26])
				(*packet).Segments = varint(vorbis[26:27])
				///79 103 103 83
				//OggS
				//log.Print("OggS version = ", packet.version)
				//log.Print("OggS header_type = ", packet.header_type)
				//log.Print("OggS serial = ", packet.serial_number)
				//log.Print("OggS sequence = ", packet.sequence)
				//log.Print("OggS serial_number = ", vorbis[14:18])
				if _DEBUGME {
					pac, _ := json.MarshalIndent(packet, "", "    ")
					log.Println("data", stringify(pac))
				}

				//log.Print("VORBIS > ", VORBIS)
				//log.Print("packet.header_type ", vorbis[5:6])
				// new song
				if packet.Header_type != 0 || alsoReadNext != 0 {

					// clean nex handler
					if alsoReadNext == 1 {
						alsoReadNext = 0
					}

					if _DEBUGME {
						log.Print("prebral ", n)
						log.Print("OggS segments = ", vorbis[26:27])
						log.Print("OggS segments = ", packet.Segments)
					}

					ARTIST := bytes.Index(vorbis[0:], []byte("ARTIST="))
					if _DEBUGME {
						log.Print("ARTIST ", ARTIST)
					}

					if ARTIST != -1 {
						(*povezava).Artist = clean(vorbis[ARTIST:])
						log.Print("ARTIST ", povezava.Artist)
					}

					TITLE := bytes.Index(vorbis[0:], []byte("TITLE="))
					if _DEBUGME {
						log.Print("TITLE ", TITLE)
					}

					if TITLE != -1 {
						(*povezava).Song = clean(vorbis[TITLE:])
						log.Print("TITLE ", povezava.Song)
					}

					// set next handler aka countionation of packet
					if ARTIST > 4000 || packet.Segments == 255 || (packet.Header_type == 4 && TITLE == -1) {
						alsoReadNext = 1
					}
					povezave[*path] = *povezava
				}

			}
		}
	}
}

func control_server_handle(conn net.Conn) {
	povezava := new(Audiocast)
	var path = conn.RemoteAddr().String()
	log.Println("client", conn.RemoteAddr(), "connected")
	for {
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			conn.Write([]byte("HTTP/1.0 500 Error\r\n\r\n"))
			checkError(err)
			return
		}

		if req.Method == "SOURCE" {
			path = req.URL.Path
			povezave[path] = *povezava
			parseOGG(conn, povezava, &path)
			break
		} else {
			conn.Write([]byte("HTTP/1.0 405 Method not allowed\r\n\r\nMethod not allowed"))
			break
		}

	}
	log.Println("client", conn.RemoteAddr(), "disconnected")
	delete(povezave, path)
	conn.Close()
}

func control_server(port string) {

	fmt.Println("icecast/shoutcast server running on port ", port)

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
	//_DEBUGME = flag.Bool("debug", false, "enable debugging")
	flag.Parse()

	//icecast server
	go control_server(toIfPort(*server_port))

	//info server
	http.HandleFunc("/", info)

	go http.ListenAndServe(toIfPort(*info_port), nil)

	// infinite loop; don't use for, this is not c
	select {}
}
