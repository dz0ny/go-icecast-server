package main

import (
	"flag"
	"fmt"
	"github.com/bradrydzewski/routes"
	"icecast"
	"net/http"
	"runtime"
	"utils"
)

var server_port = flag.Int("i", 8000, "icecast server port")
var info_port = flag.Int("c", 3000, "web server port")
var user = flag.String("u", "", "icecast server username")
var password = flag.String("p", "", "icecast server password")

func main() {

	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())
	go icecast.Server(utils.ToIfPort(*server_port), utils.Basic_auth(*user, *password))

	//info server
	mux := routes.New()
	mux.Logging = false
	mux.Post("/hook/:stream", icecast.RegisterHook)
	mux.Get("/info.json", icecast.DisplayInfo)

	http.Handle("/", mux)
	go http.ListenAndServe(utils.ToIfPort(*info_port), nil)
	fmt.Println("web server running on port ", utils.ToIfPort(*info_port))
	// infinite loop; don't use for, this is not c
	select {}

}
