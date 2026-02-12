package cluster

import (
	"encoding/binary"
	"fmt"

	"github.com/hashicorp/memberlist"
)

// Message direction byte.
const (
	dirLAN byte = 0x01
	dirWAN byte = 0x02
)

// encodeMessage builds the wire format:
//
//	[1 byte direction][2 bytes region len][region string][payload]
func encodeMessage(dir byte, region string, payload []byte) []byte {
	regionBytes := []byte(region)
	msg := make([]byte, 1+2+len(regionBytes)+len(payload))
	msg[0] = dir
	binary.BigEndian.PutUint16(msg[1:3], uint16(len(regionBytes)))
	copy(msg[3:3+len(regionBytes)], regionBytes)
	copy(msg[3+len(regionBytes):], payload)

	return msg
}

// decodeMessage parses the wire format.
func decodeMessage(data []byte) (dir byte, region string, payload []byte, err error) {
	if len(data) < 3 {
		return 0, "", nil, fmt.Errorf("message too short: %d bytes", len(data))
	}

	dir = data[0]
	regionLen := binary.BigEndian.Uint16(data[1:3])

	if len(data) < 3+int(regionLen) {
		return 0, "", nil, fmt.Errorf("message truncated: need %d bytes for region, have %d",
			regionLen, len(data)-3)
	}

	region = string(data[3 : 3+regionLen])
	payload = data[3+regionLen:]

	return dir, region, payload, nil
}

// clusterBroadcast implements memberlist.Broadcast for the TransmitLimitedQueue.
type clusterBroadcast struct {
	msg []byte
}

var _ memberlist.Broadcast = (*clusterBroadcast)(nil)

func (b *clusterBroadcast) Invalidates(other memberlist.Broadcast) bool { return false }
func (b *clusterBroadcast) Message() []byte                            { return b.msg }
func (b *clusterBroadcast) Finished()                                  {}

// newBroadcast wraps raw bytes in a memberlist.Broadcast for queue submission.
func newBroadcast(msg []byte) *clusterBroadcast {
	return &clusterBroadcast{msg: msg}
}
