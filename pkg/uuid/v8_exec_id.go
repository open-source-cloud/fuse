// Package uuid internal UUID utility functions
package uuid

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// V8ExecID generates a custom V8 UUID,
//
//	with the same timestamping as V7 but using part of the bytes in the third group for threadID
func V8ExecID(thread uint16) (string, error) {
	if thread > 0xFFF {
		return "", fmt.Errorf("value out of 12-bit range")
	}
	var uuid [16]byte

	// 1. Timestamp in ms, 6 bytes
	ms := uint64(time.Now().UnixNano() / 1e6) //nolint:gosec
	binary.BigEndian.PutUint64(uuid[0:8], ms)
	// We only use the high 6 bytes of ms (bytes 0–5)
	// bytes 6–7 overwritten below

	// 2. Set version (8) in high nibble of byte 6; low nibble with hi 4 bits of val
	uuid[6] = 0x80 | byte(thread>>8)
	// 3. Store low 8 bits of val in byte 7
	uuid[7] = byte(thread & 0xFF)

	// 4. Fill random rest (bytes 8–15)
	_, err := rand.Read(uuid[8:])
	if err != nil {
		return "", err
	}

	// 5. Set variant in byte 8
	uuid[8] &= 0x3F
	uuid[8] |= 0x80

	// 6. Format as UUID string
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(uuid[0:4]),
		binary.BigEndian.Uint16(uuid[4:6]),
		binary.BigEndian.Uint16(uuid[6:8]),
		binary.BigEndian.Uint16(uuid[8:10]),
		uuid[10:16],
	), nil
}
