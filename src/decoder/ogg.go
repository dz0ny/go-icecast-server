package ogg

import (
	"bytes"
	"errors"
)

//ogg v1 packet structure
type Packet struct {
	Version          uint32
	Header_type      uint32
	Granule_position uint64
	Sequence         uint32
	Serial_number    uint32
	Crc              uint32
	Segments         uint32
}

func NewOggpacket(ogg []byte) (Packet, error) {
	packet := new(Packet)

	if !bytes.Contains(ogg[0:4], []byte("OggS")) {
		return *packet, errors.New("Missing ogg header in bitstream")
	}

	//http://wiki.xiph.org/Ogg_Skeleton_4
	//79 103 103 83  | 0-3 header
	// 0 | 4-5 version
	// 4 | 5-6 type
	// 0 48 42 0 - 0 0 0 0 | 6-13 granule
	// 172 79 0 0 | 14-17 serial_number
	// 241 0 0 0 | 18-21 sequence

	(*packet).Version = Varint32(ogg[4:5])
	(*packet).Header_type = Varint32(ogg[5:6])
	(*packet).Granule_position = Varint64(ogg[6:14])
	(*packet).Serial_number = Varint32(ogg[14:18])
	(*packet).Sequence = Varint32(ogg[18:22])
	(*packet).Crc = Varint32(ogg[22:26])
	(*packet).Segments = Varint32(ogg[26:27])

	return *packet, nil
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
