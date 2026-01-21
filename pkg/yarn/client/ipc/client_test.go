package ipc

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDelimited(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"short data", []byte{0x01, 0x02, 0x03}},
		{"long data", make([]byte, 300)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			encoded := encodeDelimited(c.data)

			// 读取前缀长度
			length, n := binary.Uvarint(encoded)
			assert.Equal(t, uint64(len(c.data)), length)
			assert.Equal(t, len(encoded), n+len(c.data))
			assert.Equal(t, c.data, encoded[n:])
		})
	}
}
