package icecast

import (
	"bufio"
	"decoder/ogg"
	"fmt"
	"github.com/bradrydzewski/routes"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"utils"
)

//max 16 clients
var povezave = make(map[string]Audiocast, 16)

type Audiocast struct {
	Artist      string   `json:"artist"`
	Song        string   `json:"song"`
	Encoder     string   `json:"encoder"`
	Name        string   `json:"station-name"`
	Description string   `json:"station-description"`
	Audio       string   `json:"station-info"`
	Type        string   `json:"content-type"`
	WebHooks    []string `json:"hooks"`
}

func Server(port string, basic_auth string) {

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
		go handle_request(conn, basic_auth)

	}

}

func handle_request(conn net.Conn, basic_auth string) {

	log.Println("client", conn.RemoteAddr(), "connected")

	for {
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			io.WriteString(conn, "HTTP/1.0 500 Error\r\n\r\n")
			utils.CheckError(err)
			break
		}
		//chech for authorization
		auth := req.Header.Get("Authorization")
		if len(auth) != 0 && len(basic_auth) > 1 {
			if !strings.Contains(auth, "Basic "+basic_auth) {
				io.WriteString(conn, "HTTP/1.0 402 Not Authorized\r\n\r\nDon't poke here!")
				break
			}
		} else if len(basic_auth) > 1 {
			io.WriteString(conn, "HTTP/1.0 401 Not Authorized\r\nWWW-Authenticate: Basic realm=\"Icecast Server\"r\n")
			break
		}
		if req.Method == "SOURCE" {
			parseIcecast(conn, req)
			break
		} else if req.Method == "GET" && req.URL.Path == "/admin/metadata" {
			parseMetadataUpdate(conn, req)
			break
		} else {
			io.WriteString(conn, "HTTP/1.0 405 Method not allowed\r\n\r\nMethod not allowed")
			break
		}
	}

	log.Println("client", conn.RemoteAddr(), "disconnected")

	conn.Close()
}

//icecast2 update
func parseMetadataUpdate(conn net.Conn, req *http.Request) {
	//example GET /admin/metadata?mode=updinfo&mount=/mixx&song=Test%20%2d%20Test
	mode := req.URL.Query().Get("mode")
	mount := req.URL.Query().Get("mount")
	song := req.URL.Query().Get("song")

	if povezava, ok := povezave[mount]; mode == "updinfo" && ok && len(song) > 0 && strings.Contains(song, "-") {
		s := strings.Split(song, " - ")
		io.WriteString(conn, "HTTP/1.0 200 OK\r\n\r\nUpdated")
		povezava.Artist = s[0]
		povezava.Song = s[1]
		povezave[mount] = povezava
	} else {
		io.WriteString(conn, "HTTP/1.0 404 Not Found\r\n\r\n")
	}
	return
}

func parseIcecast(conn net.Conn, req *http.Request) {

	povezava := new(Audiocast)
	stream := req.URL.Path[1:]
	//check for streams limit
	if len(povezave) >= 16 {
		io.WriteString(conn, "HTTP/1.0 405 Too many streams\r\n\r\nToo many streams")
		return
	}

	povezava.Type = req.Header.Get("Content-Type")
	povezava.Name = req.Header.Get("Ice-Name")
	povezava.Description = req.Header.Get("Ice-Description")
	povezava.Audio = req.Header.Get("Ice-Audio-Info")
	povezave[stream] = *povezava

	//icecast 1 update
	io.WriteString(conn, "HTTP/1.0 200 OK\r\n\r\n")
	for {
		// CALLS FOR REWRITE WITH CHANNELS
		var data [1024 * 64]byte
		oggAdd := 0

		read, err := conn.Read(data[0:])
		if err != nil {
			break
		}

		for oggAdd >= 0 {
			packet, errp := ogg.NewOggpacket(data[oggAdd:read], &oggAdd)
			if errp != nil {
				break
			}
			if packet.Info != nil && errp == nil {

				//Update info.json
				//if same artis was played before
				// encoder will only update track
				if packet.Info.Artist != "" {
					povezava.Artist = packet.Info.Artist
				}

				if packet.Info.Song != "" {
					povezava.Song = packet.Info.Song
				}

				povezava.Encoder = packet.Info.Encoder
				log.Println("InfoPacket", povezava)
				povezave[stream] = *povezava

				//update all web hooks
				for _, hook_url := range povezava.WebHooks {
					update := new(url.Values)
					update.Add("artist", povezava.Artist)
					update.Add("song", povezava.Song)
					status, err := http.PostForm(hook_url, *update)
					log.Println(status, err)
				}

			}
		}

	}

	//utils.Cleanup
	delete(povezave, req.URL.Path)
	return

}

func RegisterHook(res http.ResponseWriter, req *http.Request) {

	params := req.URL.Query()
	stream := params.Get(":stream")

	hook_string := req.FormValue("callback")

	povezava, ok := povezave[stream]

	if ok && hook_string != "" {

		povezava.WebHooks = append(povezava.WebHooks, hook_string)
		povezave[stream] = povezava

		io.WriteString(res, "OK subscribed: "+hook_string+" to "+stream)
	} else {
		io.WriteString(res, "FAILED subscribe: "+hook_string+" to "+stream)
	}
	return
}

func DisplayInfo(res http.ResponseWriter, req *http.Request) {
	routes.ServeJson(res, &povezave)
	return
}
