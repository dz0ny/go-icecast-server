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
	Song   string
	Artist string
}

func parseForComments(data []byte, song *Meta) {

	ARTIST := bytes.Index(data[0:], []byte("ARTIST="))
	if ARTIST != -1 {
		song.Artist = utils.Clean(data[ARTIST:])
	} else {
		song.Artist = ""
	}

	TITLE := bytes.Index(data[0:], []byte("TITLE="))
	if TITLE != -1 {
		song.Song = utils.Clean(data[TITLE:])
	} else {
		song.Song = ""
	}

}

func ParsePacket(data []byte) [][]byte {
	ogg := bytes.Split(data[0:], []byte(OggHeader))
	return ogg
}

//returns new parsed ogg packet or err if it's not ogg packet
func NewOggpacket(ogg []byte, skipCheck bool) (Packet, error) {
	packet := new(Packet)

	if !skipCheck {
		if !bytes.Contains(ogg[0:4], []byte(OggHeader)) {
			return *packet, errors.New("Missing ogg header in bitstream")
		}
	}

	packet.stream_structure_version = Varint32(ogg[4:5])
	packet.header_type_flag = Varint32(ogg[5:6])
	packet.absolute_granule_position = Varint64(ogg[6:14])
	packet.stream_serial_number = Varint32(ogg[14:18])
	packet.page_sequence_no = Varint32(ogg[18:22])
	packet.page_checksum = Varint32(ogg[22:26])
	packet.page_segments = Varint32(ogg[26:27])

	if packet.header_type_flag != 0 {

		meta := new(Meta)
		parseForComments(ogg[27:], meta)

		if len(meta.Artist) > 0 && len(meta.Song) > 0 {
			log.Println("song", meta)
			packet.Info = meta
		}
	}

	return *packet, nil
}

// Fix page header based on sent packets
func UpdatePageSequence(data []byte) (packet []byte) {
	return data
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
