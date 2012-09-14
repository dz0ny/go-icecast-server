package ogg

import (
	"bytes"
	"errors"
	"log"
	"utils"
)

const (
	OggHeader = "OggS"
)

//ogg v1 packet structure see http://xiph.org/ogg/doc/framing.html
type Packet struct {
	stream_structure_version  uint32
	header_type_flag          uint32
	absolute_granule_position uint64
	stream_serial_number      uint32
	page_sequence_no          uint32
	page_checksum             uint32
	page_segments             uint32
	segment_table             uint32
	Info                      *Meta
}

type Meta struct {
	Song    string
	Artist  string
	Encoder string
}

func parseForComments(data []byte, meta *Meta) {

	ARTIST := bytes.Index(data[0:], []byte("ARTIST="))
	if ARTIST != -1 {
		meta.Artist = utils.Clean(data[ARTIST:])
	} else {
		meta.Artist = ""
	}

	TITLE := bytes.Index(data[0:], []byte("TITLE="))
	if TITLE != -1 {
		meta.Song = utils.Clean(data[TITLE:])
	} else {
		meta.Song = ""
	}

	ENCODER := bytes.Index(data[0:], []byte("ENCODER="))
	if ENCODER != -1 {
		meta.Encoder = utils.Clean(data[ENCODER:])
	} else {
		meta.Encoder = ""
	}

}

//returns new parsed ogg packet or err if it's not ogg packet
func NewOggpacket(ogg []byte, startAddress int) (Packet, error, int) {
	packet := new(Packet)
	var headerLoc int
	if startAddress > 0 {
		headerLoc = startAddress
	} else {
		headerLoc = bytes.Index(ogg[0:], []byte(OggHeader))
	}

	if headerLoc == -1 {
		return *packet, errors.New("Missing ogg header in bitstream"), -1
	}

	packet.stream_structure_version = Varint32(ogg[headerLoc+4 : 5+headerLoc])
	packet.header_type_flag = Varint32(ogg[headerLoc+5 : 6+headerLoc])
	packet.absolute_granule_position = Varint64(ogg[headerLoc+6 : 14+headerLoc])
	packet.stream_serial_number = Varint32(ogg[headerLoc+14 : 18+headerLoc])
	packet.page_sequence_no = Varint32(ogg[headerLoc+18 : 22+headerLoc])
	packet.page_checksum = Varint32(ogg[headerLoc+22 : 26+headerLoc])
	packet.page_segments = Varint32(ogg[headerLoc+26 : 27+headerLoc])

	if packet.header_type_flag != 0 {

		meta := new(Meta)
		parseForComments(ogg[27:], meta)

		if len(meta.Artist) > 0 && len(meta.Song) > 0 {
			log.Println("song", meta)
			packet.Info = meta
		}
	}
	//find next and return position
	headerLoc = bytes.Index(ogg[27+headerLoc:], []byte(OggHeader))
	return *packet, nil, headerLoc
}

// convert []bytes to uint32
func Varint32(slice []byte) uint32 {
	number := uint32(slice[0])
	shift := uint(8)

	for i := 1; i < len(slice); i++ {

		number |= uint32(slice[i]) << shift
		shift *= 2
	}
	return number
}

// convert []bytes to uint32
func Varint64(slice []byte) uint64 {
	number := uint64(slice[0])
	shift := uint(8)

	for i := 1; i < len(slice); i++ {

		number |= uint64(slice[i]) << shift
		shift *= 2
	}
	return number
}
