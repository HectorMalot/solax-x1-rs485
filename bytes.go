package solaxx1rs485

// This file contains various utilities to work with
// binary bodies. e.g. to convert uint(8,16,32) to []bytes

func bytesFromUint16(in uint16) []byte {
	return []byte{uint8(in / 256), uint8(in % 256)}
}

// returns a uint16 from the provided byte array (length 2)
func uint16FromBytes(in [2]byte) uint16 {
	return uint16(in[0])*256 + uint16(in[1])
}

// returns a uint32 from the provided byte array (length 4)
func uint32FromBytes(in [4]byte) uint32 {
	return uint32(in[0])*256*256*256 + uint32(in[1])*256*256 + uint32(in[2])*256 + uint32(in[3])
}
