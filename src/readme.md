#How it works

Streamer sends data in ogg packets, called slice (observed 56-4600 bytes), bellow is how they look
    //http://wiki.xiph.org/Ogg_Skeleton_4
    //79 103 103 83  | 0-3 header
    // 0 | 4-5 version
    // 4 | 5-6 type
    // 0 48 42 0 - 0 0 0 0 | 6-13 granule
    // 172 79 0 0 | 14-17 serial_number
    // 241 0 0 0 | 18-21 sequence

    packet.Version = Varint32(ogg[4:5])
    packet.Header_type = Varint32(ogg[5:6])
    packet.Granule_position = Varint64(ogg[6:14])
    packet.Serial_number = Varint32(ogg[14:18])
    packet.Sequence = Varint32(ogg[18:22])
    packet.Crc = Varint32(ogg[22:26])
    packet.Segments = Varint32(ogg[26:27])

Most important are:

 - header type, (start of song int(2) > BOS, end of song int(4) >EOS  and continuation of song int(0))
 - sequence, stream consist of multiple songs this number tels us which slice of song has been received
 - segments this contain number of comment fields
 - comments contain data both human readable data and audio info data (channles, bitrate of encoding)

BOS is only transmitted in first packet, and you must store it for later use for client (player).

Not all streamers send song title and artist in comments filed, for example mixxx uses http get call to /admin/metadata

The trickiest part is, when you have to send data to client (player). On  new connection you have to reconstruct ogg packet,
most notably sequence(which must start from zero) and segments(if packet is either BOS or EOS). On BOS sequence, segments must also contain
audio info data, which is used by decoder on player side. 