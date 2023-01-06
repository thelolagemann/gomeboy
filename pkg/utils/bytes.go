package utils

func BytesToUint16(upper, lower uint8) uint16 {
	return uint16(upper)<<8 ^ uint16(lower)
}

func Uint16ToBytes(value uint16) (upper, lower uint8) {
	return uint8(value >> 8), uint8(value & 0xFF)
}
