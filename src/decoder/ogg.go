package ogg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"utils"
)

const (
	OggHeader = "OggS"
)

//ogg v1 packet structure see http://xiph.org/ogg/doc/framing.html
type Packet struct {
	Stream_structure_version  uint8
	Header_type_flag          uint8
	Absolute_granule_position uint64
	Stream_serial_number      uint32
	Page_sequence_no          uint32
	Page_checksum             uint32
	Page_segments             uint8
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

func FindOgg(data []byte) int {
	return bytes.Index(data[0:], []byte(OggHeader))
}

//returns new parsed ogg packet or err if it's not ogg packet
func NewOggpacket(ogg []byte, startAddress int) (Packet, error, int) {
	packet := new(Packet)
	var headerLoc int

	if startAddress > 0 {
		headerLoc = startAddress
	} else {
		headerLoc = FindOgg(ogg[0:])
	}

	if headerLoc == -1 {
		return *packet, errors.New("Missing ogg header in bitstream"), -1
	}

	packet.Stream_structure_version = uint8(ogg[headerLoc+4])
	packet.Header_type_flag = uint8(ogg[headerLoc+5])
	packet.Absolute_granule_position = Varint64(ogg[headerLoc+6 : 14+headerLoc])
	packet.Stream_serial_number = Varint32(ogg[headerLoc+14 : 18+headerLoc])
	packet.Page_sequence_no = Varint32(ogg[headerLoc+18 : 22+headerLoc])
	packet.Page_checksum = Varint32(ogg[headerLoc+22 : 26+headerLoc])
	packet.Page_segments = uint8(ogg[headerLoc+26])

	if packet.Header_type_flag != 0 {

		meta := new(Meta)
		parseForComments(ogg[27:], meta)

		if len(meta.Artist) > 0 && len(meta.Song) > 0 {
			log.Println("song", meta)
			packet.Info = meta
		}
	}

	return *packet, nil, headerLoc
}

// convert []bytes to uint32
func Varint32(slice []byte) uint32 {
	l := len(slice)
	if l == 1 {
		return uint32(slice[0])
	}
	return binary.LittleEndian.Uint32(slice[0:])
}

// convert []bytes to uint32
func Varint64(slice []byte) uint64 {
	l := len(slice)
	if l == 1 {
		return uint64(slice[0])
	}
	return binary.LittleEndian.Uint64(slice[0:])
}
