package cluster

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeMessage(t *testing.T) {
	tests := []struct {
		name    string
		dir     byte
		region  string
		payload []byte
	}{
		{
			name:    "LAN message",
			dir:     dirLAN,
			region:  "us-east-1",
			payload: []byte("hello world"),
		},
		{
			name:    "WAN message",
			dir:     dirWAN,
			region:  "eu-west-1",
			payload: []byte(`{"key":"value"}`),
		},
		{
			name:    "empty payload",
			dir:     dirLAN,
			region:  "ap-south-1",
			payload: []byte{},
		},
		{
			name:    "binary payload",
			dir:     dirWAN,
			region:  "us-west-2",
			payload: []byte{0x00, 0x01, 0xFF, 0xFE},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeMessage(tt.dir, tt.region, tt.payload)
			dir, region, payload, err := decodeMessage(encoded)

			require.NoError(t, err)
			require.Equal(t, tt.dir, dir)
			require.Equal(t, tt.region, region)
			require.Equal(t, tt.payload, payload)
		})
	}
}

func TestDecodeMessage_TooShort(t *testing.T) {
	_, _, _, err := decodeMessage([]byte{0x01})
	require.Error(t, err)
}

func TestDecodeMessage_Truncated(t *testing.T) {
	// Header says region is 10 bytes but only 2 bytes follow
	data := []byte{0x01, 0x00, 0x0A, 'u', 's'}
	_, _, _, err := decodeMessage(data)
	require.Error(t, err)
}
