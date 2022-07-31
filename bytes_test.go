package solaxx1rs485

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytesFromUint16(t *testing.T) {
	in := uint16(0x0402)
	fmt.Println(in)
	out := bytesFromUint16(in)
	require.Equal(t, uint8(0x4), out[0])
	require.Equal(t, uint8(0x2), out[1])
}
