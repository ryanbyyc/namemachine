package namemachine

import (
	cryptoRand "crypto/rand"
)

/**
 * base32 is the lowercase rfc4648 alphabet used for slugs
 * letters a to z then digits two to seven
 * compact and url friendly
 */
var base32 = []byte("abcdefghijklmnopqrstuvwxyz234567")

/**
 * randomSlugInto appends a base32 slug of length n into dst
 * uses crypto strong randomness and falls back to a safe filler on error
 * zero heap when caller provides capacity
 * @param dst []byte destination buffer provided by caller
 * @param n int desired slug length
 * @return []byte the destination buffer with slug appended
 */
func randomSlugInto(dst []byte, n int) []byte {
	if n <= 0 {
		return dst
	}

	// read randomness in a small fixed buffer for speed and simplicity
	var buf [16]byte
	i := 0

	for i < n {
		// try to fill the buffer with crypto randomness
		if _, err := cryptoRand.Read(buf[:]); err != nil {
			// on failure fill the remainder with the first alphabet symbol
			for i < n {
				dst = append(dst, base32[0])
				i++
			}
			break
		}

		// map each random byte to an alphabet index using modulo
		for _, b := range buf {
			dst = append(dst, base32[int(b)%len(base32)])
			i++
			if i >= n {
				break
			}
		}
	}
	return dst
}
