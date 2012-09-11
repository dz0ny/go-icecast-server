package ogg

//ogg v1 packet structure
type OggPacket struct {
	Version          uint32
	Header_type      uint32
	Granule_position uint64
	Sequence         uint32
	Serial_number    uint32
	Crc              uint32
	Segments         uint32
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
