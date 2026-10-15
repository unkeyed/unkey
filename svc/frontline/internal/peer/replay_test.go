package peer

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestReplayBody_NeverSendsUnrecordedBytesBeforeAck guards the retry rule:
// before the relay acknowledges its dial, no attempt may read a byte that a
// later attempt could not replay. After the ack, the rest streams through.
func TestReplayBody_NeverSendsUnrecordedBytesBeforeAck(t *testing.T) {
	const limit = 8
	src := []byte("0123456789abcdefghij")
	body := NewReplayBody(bytes.NewReader(src), limit)

	first, err := body.Attempt()
	require.NoError(t, err)
	got := make([]byte, 5)
	n, err := io.ReadFull(first, got)
	require.NoError(t, err)
	require.Equal(t, "01234", string(got[:n]))

	second, err := body.Attempt()
	require.NoError(t, err)
	_, err = first.Read(got)
	require.ErrorIs(t, err, errAttemptOver)

	prefix := make([]byte, limit)
	_, err = io.ReadFull(second, prefix)
	require.NoError(t, err)
	require.Equal(t, "01234567", string(prefix))

	blocked := make(chan []byte)
	go func() {
		rest, _ := io.ReadAll(second)
		blocked <- rest
	}()
	select {
	case <-blocked:
		t.Fatal("read past the replay limit before the ack")
	case <-time.After(50 * time.Millisecond):
	}

	second.(*replayAttempt).Ack()
	require.Equal(t, "89abcdefghij", string(<-blocked))

	_, err = body.Attempt()
	require.ErrorIs(t, err, errAcked)
}
