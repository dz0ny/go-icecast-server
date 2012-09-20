# Icecast/Shoutcast Proxy

  It can be used to display data from icecast stream broadcaster, which is normally encoded in ogg/vorbis packet.

## Simple API

 - GET  /latest.json - List of all connected streamer, with latest data about song being played
 - POST /hook/{stream_name} - Register web hook for this stream, when broadcaster(DJ Traktor, Mixx ...) updates song info.

## CLI
  
  bin/server -h
  Usage of bin/server:
    -c=3000: web server port
    -i=8000: icecast server port
    -p="": icecast server password
    -u="": icecast server username

