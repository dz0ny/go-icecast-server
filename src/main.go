package main

import (
	"bufio"
	"bytes"
	"decoder/ogg"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"utils"
)

type Audiocast struct {
	Artist      string `json:"artist"`
	Song        string `json:"song"`
	Name        string `json:"station-name"`
	Description string `json:"station-description"`
	Audio       string `json:"station-info"`
	Type        string `json:"content-type"`
}

//max 16 clients
var povezave = make(map[string]Audiocast, 16)
var _DEBUGME bool

//icecast2 update
func parseMetadataUpdate(conn net.Conn, req *http.Request) {
	//example GET /admin/metadata?mode=updinfo&mount=/mixx&song=Test%20%2d%20Test
	mode := req.URL.Query().Get("mode")
	mount := req.URL.Query().Get("mount")
	song := req.URL.Query().Get("song")
	if _DEBUGME {
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
func parseOGG(conn net.Conn, req *http.Request) {
	conn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	red := bufio.NewReader(conn)
	var vorbis [4096]byte
	alsoReadNext := 0
	for {
		n, err := red.Read(vorbis[0:])
		if err != nil {
			//utils.CheckError(err) //eof
			break
		}

		if n > 0 {

			//vorbis data packet
			if bytes.Contains(vorbis[0:4], []byte("OggS")) {
				packet := new(ogg.OggPacket)

				//http://wiki.xiph.org/Ogg_Skeleton_4
				//79 103 103 83  | 0-3 header
				// 0 | 4-5 version
				// 4 | 5-6 type
				// 0 48 42 0 - 0 0 0 0 | 6-13 granule
				// 172 79 0 0 | 14-17 serial_number
				// 241 0 0 0 | 18-21 sequence

				(*packet).Version = ogg.Varint32(vorbis[4:5])
				(*packet).Header_type = ogg.Varint32(vorbis[5:6])
				(*packet).Granule_position = ogg.Varint64(vorbis[6:14])
				(*packet).Serial_number = ogg.Varint32(vorbis[14:18])
				(*packet).Sequence = ogg.Varint32(vorbis[18:22])
				(*packet).Crc = ogg.Varint32(vorbis[22:26])
				(*packet).Segments = ogg.Varint32(vorbis[26:27])

				if packet.Header_type != 0 || alsoReadNext != 0 {

					povezava, _ := povezave[req.URL.Path]
					// utils.Clean nex handler
					if alsoReadNext == 1 {
						alsoReadNext = 0
					}
					if _DEBUGME {
						pac, _ := json.MarshalIndent(packet, "", "    ")
						log.Println("data", utils.Stringify(pac))
					}
					ARTIST := bytes.Index(vorbis[0:], []byte("ARTIST="))
					if _DEBUGME {
						log.Print("ARTIST ", ARTIST)
					}

					if ARTIST != -1 {
						(povezava).Artist = utils.Clean(vorbis[ARTIST:])
						log.Print("ARTIST ", povezava.Artist)
					}

					TITLE := bytes.Index(vorbis[0:], []byte("TITLE="))
					if _DEBUGME {
						log.Print("TITLE ", TITLE)
					}

					if TITLE != -1 {
						(povezava).Song = utils.Clean(vorbis[TITLE:])
						log.Print("TITLE ", povezava.Song)
					}

					// set next handler aka countionation of packet
					if ARTIST > 4000 || packet.Segments == 255 || (packet.Header_type != 0 && TITLE == -1) {
						alsoReadNext = 1
					}
					povezave[req.URL.Path] = povezava
				}

			}
		}
	}
}

func control_server_handle(conn net.Conn, basic_auth string) {
	povezava := new(Audiocast)

	if _DEBUGME {
		log.Println("client", conn.RemoteAddr(), "connected")
	}

	for {
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			conn.Write([]byte("HTTP/1.0 500 Error\r\n\r\n"))
			utils.CheckError(err)
			break
		}
		if _DEBUGME {
			log.Println("req", req)
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
			parseOGG(conn, req)

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
	if _DEBUGME {
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

func main() {

	//cli
	var server_port = flag.Int("port", 8000, "icecast server port")
	var info_port = flag.Int("web_port", 3000, "web server port")
	var user = flag.String("username", "", "icecast server username")
	var password = flag.String("password", "", "icecast server password")
	flag.BoolVar(&_DEBUGME, "debug", false, "enable debugging")
	flag.Parse()

	//icecast server
	go control_server(utils.ToIfPort(*server_port), utils.Basic_auth(*user, *password))

	//info server
	http.HandleFunc("/info.json", info)
	http.Handle("/", http.FileServer(http.Dir("web/public")))

	go http.ListenAndServe(utils.ToIfPort(*info_port), nil)

	// infinite loop; don't use for, this is not c
	select {}
}
