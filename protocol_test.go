package solaxx1rs485

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChecksum(t *testing.T) {
	t.Run("Simple sum", func(t *testing.T) {
		body := []byte{1, 2, 3, 4, 5, 6}
		cs := checksum(body)
		require.Equal(t, uint16(21), cs)
	})

	t.Run("Sum above 256 gives correct value", func(t *testing.T) {
		body := []byte{0, 0, 255, 1}
		cs := checksum(body)
		require.Equal(t, uint16(0x100), cs)
	})

	t.Run("Overflow gives correct value", func(t *testing.T) {
		body := []byte{255}
		for i := 0; i < 256; i++ {
			body = append(body, 255)
		}
		cs := checksum(body)
		require.Equal(t, uint16(0xFFFF), cs)

		body = append(body, 2)
		cs = checksum(body)
		require.Equal(t, uint16(0x0001), cs)
	})
}

func FuzzParsePacket(f *testing.F) {
	f.Add([]byte{0x44})
	f.Fuzz(func(t *testing.T, data []byte) {
		p := DefaultPacket()
		p.Data = data
		body, err := p.Bytes()
		if err != nil {
			if err == ErrMaxDataSizeExceeded {
				return
			}
			t.Fatal(err)
		}

		parsed, err := ParsePacket(body)
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, parsed.Data, p.Data)
	})
}
