package utils

import (
	"log"
	"strconv"
)

func getUintFromHex(b byte) uint8 {
	i, err := strconv.ParseUint((string)(b), 16, 8)
	if err != nil {
		log.Printf("Error parsing %v: %v", b, err)
		return 0
	}
	return uint8(i)
}

func ParseHexColor(s string) (r, g, b, a uint8) {
	if len(s) >= 7 && s[0] == '#' {
		r1 := getUintFromHex(s[1]) * 16
		r2 := getUintFromHex(s[2])
		r = r1 + r2
		g1 := getUintFromHex(s[3]) * 16
		g2 := getUintFromHex(s[4])
		g = g1 + g2
		b1 := getUintFromHex(s[5]) * 16
		b2 := getUintFromHex(s[6])
		b = b1 + b2
		if len(s) == 9 {
			a1 := getUintFromHex(s[7]) * 16
			a2 := getUintFromHex(s[8])
			a = a1 + a2
			return r, g, b, a
		}
		return r, g, b, 255
	}
	return 0, 0, 0, 255
}

