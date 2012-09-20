# Icecast info server

  It can be used to display data from icecast stream broadcaster, which is normally encoded in ogg/vorbis packet.

### Simple API

 - GET  /latest.json - List of all connected streamers, with latest data about song being streamed
 - POST /hook/{stream_name} - Register web hook for this stream, when broadcaster(DJ Traktor, Mixx ...) updates song info.
 - SOURCE /{stream_name} - Send data to this URI using your broadcast software

### CLI
  
    bin/server -h
    Usage of bin/server:
      -c=3000: web server port
      -i=8000: icecast server port
      -p="": icecast server password 
      -u="": icecast server username (for most broadcasting software defaults to source)

### Performance (stream 192kbits 44,1khz; 3,2GHz AMD Phenom2, 12GB RAM)

    Command exited with non-zero status 2
      Command being timed: "bin/server -c=3002 -i=8001"
      User time (seconds): 0.16
      System time (seconds): 0.05
      Percent of CPU this job got: 0%
      Elapsed (wall clock) time (h:mm:ss or m:ss): 5:57.17
      Average shared text size (kbytes): 0
      Average unshared data size (kbytes): 0
      Average stack size (kbytes): 0
      Average total size (kbytes): 0
      Maximum resident set size (kbytes): 14016
      Average resident set size (kbytes): 0
      Major (requiring I/O) page faults: 0
      Minor (reclaiming a frame) page faults: 988
      Voluntary context switches: 3055
      Involuntary context switches: 1858
      Swaps: 0
      File system inputs: 0
      File system outputs: 0
      Socket messages sent: 0
      Socket messages received: 0
      Signals delivered: 0
      Page size (bytes): 4096
      Exit status: 2

