package server

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/stretchr/testify/require"
)

func TestTCPPipeliningDrainsBeforeClose(t *testing.T) {
	for _, halfClose := range []bool{false, true} {
		name, count := "query-limit", dnswire.MaxTCPQueries
		if halfClose {
			name, count = "client-half-close", 32
		}
		t.Run(name, func(t *testing.T) {
			release := make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			_, conn := startTCPTest(t, dnswire.HandlerFunc(func(ctx context.Context, w dnswire.ResponseWriter, request *dnswire.Msg) {
				if request.ID == 0 {
					select {
					case <-release:
					case <-ctx.Done():
						return
					}
				}
				tcpTestReply(t, w, request)
			}))
			t.Cleanup(unblock)
			var frames []byte
			for id := range count {
				query := dnswire.NewMsg("example.org.", dnswire.TypeA)
				query.ID = uint16(id)
				frames = append(frames, tcpTestFrame(t, query)...)
			}
			n, err := conn.Write(frames)
			require.NoError(t, err)
			require.Len(t, frames, n)
			if halfClose {
				require.NoError(t, conn.CloseWrite())
			}
			seen := make(map[uint16]bool)
			for range count - 1 {
				reply := readTCPTestReply(t, conn)
				require.NotZero(t, reply.ID, "later queries must finish while request zero is blocked")
				require.Less(t, int(reply.ID), count)
				require.False(t, seen[reply.ID], "duplicate reply")
				seen[reply.ID] = true
			}
			unblock()
			require.Zero(t, readTCPTestReply(t, conn).ID)
			require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
			_, err = conn.Read(make([]byte, 1))
			require.ErrorIs(t, err, io.EOF)
		})
	}
}

func TestTCPValidationAndFraming(t *testing.T) {
	var calls atomic.Int32
	_, conn := startTCPTest(t, dnswire.HandlerFunc(func(_ context.Context, w dnswire.ResponseWriter, request *dnswire.Msg) {
		calls.Add(1)
		tcpTestReply(t, w, request)
	}))
	ignored := dnswire.NewMsg("ignored.example.", dnswire.TypeA)
	ignored.ID, ignored.Response = 1, true
	unsupported := dnswire.NewMsg("unsupported.example.", dnswire.TypeA)
	unsupported.ID, unsupported.Opcode = 2, 15
	noQuestion := new(dnswire.Msg)
	noQuestion.ID = 3
	refused := dnswire.NewMsg("refused.example.", dnswire.TypeRRSIG)
	refused.ID = 4
	accepted := dnswire.NewMsg("example.org.", dnswire.TypeA)
	accepted.ID = 5
	var frames []byte
	for _, query := range []*dnswire.Msg{ignored, unsupported, noQuestion, refused, accepted} {
		frames = append(frames, tcpTestFrame(t, query)...)
	}
	for _, chunk := range [][]byte{frames[:1], frames[1:7], frames[7:]} {
		n, err := conn.Write(chunk)
		require.NoError(t, err)
		require.Len(t, chunk, n)
	}
	require.NoError(t, conn.CloseWrite())
	want := map[uint16]uint16{2: dnswire.RcodeNotImplemented, 3: dnswire.RcodeFormatError, 4: dnswire.RcodeRefused, 5: dnswire.RcodeSuccess}
	for range 4 {
		reply := readTCPTestReply(t, conn)
		code, exists := want[reply.ID]
		require.True(t, exists, "unexpected reply ID %d", reply.ID)
		require.Equal(t, code, reply.Rcode)
		delete(want, reply.ID)
	}
	_, err := conn.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF)
	require.EqualValues(t, 1, calls.Load())
}

func TestTCPIgnoredRequestsStillCountTowardLimit(t *testing.T) {
	_, conn := startTCPTest(t, dnswire.HandlerFunc(func(context.Context, dnswire.ResponseWriter, *dnswire.Msg) {
		t.Error("ignored response reached handler")
	}))
	query := dnswire.NewMsg("example.org.", dnswire.TypeA)
	query.Response = true
	frame := tcpTestFrame(t, query)
	for range dnswire.MaxTCPQueries {
		_, err := conn.Write(frame)
		require.NoError(t, err)
	}
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
	_, err := conn.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF)
}

func TestTCPClosesOnMalformedFrame(t *testing.T) {
	for _, tc := range []struct {
		name      string
		frame     []byte
		halfClose bool
	}{
		{name: "short-header", frame: []byte{0, 1, 0}, halfClose: false},
		{name: "truncated-payload", frame: []byte{0, 40, 0, 0}, halfClose: true},
		{name: "truncated-prefix", frame: []byte{0}, halfClose: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, conn := startTCPTest(t, dnswire.HandlerFunc(func(context.Context, dnswire.ResponseWriter, *dnswire.Msg) {
				t.Error("malformed frame reached handler")
			}))
			_, err := conn.Write(tc.frame)
			require.NoError(t, err)
			if tc.halfClose {
				require.NoError(t, conn.CloseWrite())
			}
			require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
			_, err = conn.Read(make([]byte, 1))
			require.Error(t, err)
			if networkErr, ok := errors.AsType[net.Error](err); ok {
				require.False(t, networkErr.Timeout())
			}
		})
	}
}

func TestTCPShutdownDrainsOrCancelsRequests(t *testing.T) {
	for _, force := range []bool{false, true} {
		name := "drain"
		if force {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			server, conn := startTCPTest(t, dnswire.HandlerFunc(func(ctx context.Context, w dnswire.ResponseWriter, request *dnswire.Msg) {
				close(entered)
				select {
				case <-release:
					tcpTestReply(t, w, request)
				case <-ctx.Done():
					if !force {
						t.Error("graceful shutdown cancelled an accepted request")
					}
				}
			}))
			t.Cleanup(unblock)
			_, err := conn.Write(tcpTestFrame(t, dnswire.NewMsg("example.org.", dnswire.TypeA)))
			require.NoError(t, err)
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("handler did not start")
			}
			timeout := time.Second
			if force {
				timeout = 30 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(t.Context(), timeout)
			t.Cleanup(cancel)
			done := make(chan error, 1)
			go func() { done <- server.Shutdown(ctx) }()
			require.Eventually(t, func() bool {
				server.mu.Lock()
				defer server.mu.Unlock()
				return server.closing
			}, time.Second, time.Millisecond)
			if !force {
				unblock()
				require.Equal(t, uint16(dnswire.RcodeSuccess), readTCPTestReply(t, conn).Rcode)
			}
			select {
			case err := <-done:
				if force {
					require.ErrorIs(t, err, context.DeadlineExceeded)
				} else {
					require.NoError(t, err)
				}
			case <-time.After(time.Second):
				t.Fatal("shutdown did not finish")
			}
		})
	}
}

func TestTCPShutdownInterruptsIdleReaders(t *testing.T) {
	server, conn := startTCPTest(t, dnswire.HandlerFunc(func(context.Context, dnswire.ResponseWriter, *dnswire.Msg) {
		t.Error("idle connection reached handler")
	}))
	require.Eventually(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return len(server.peers) == 1
	}, time.Second, time.Millisecond)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancel)
	require.NoError(t, server.Shutdown(ctx))
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
	_, err := conn.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF)
}

func TestTCPShutdownInterruptsBlockedWriter(t *testing.T) {
	entered, written := make(chan struct{}), make(chan error, 1)
	server, conn := startTCPTest(t, dnswire.HandlerFunc(func(_ context.Context, w dnswire.ResponseWriter, _ *dnswire.Msg) {
		close(entered)
		_, err := w.Write(make([]byte, 16*1024*1024))
		written <- err
	}))
	_, err := conn.Write(tcpTestFrame(t, dnswire.NewMsg("example.org.", dnswire.TypeA)))
	require.NoError(t, err)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	t.Cleanup(cancel)
	require.ErrorIs(t, server.Shutdown(ctx), context.DeadlineExceeded)
	select {
	case err := <-written:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("socket close did not interrupt blocked write")
	}
}

func TestTCPListenerFailureCancelsActiveRequests(t *testing.T) {
	entered := make(chan struct{})
	server, conn := startTCPServer(t, dnswire.HandlerFunc(func(ctx context.Context, _ dnswire.ResponseWriter, _ *dnswire.Msg) {
		close(entered)
		<-ctx.Done()
	}), func(err error) {
		require.ErrorIs(t, err, net.ErrClosed, "losing the listener without Shutdown must fail Serve")
	})
	_, err := conn.Write(tcpTestFrame(t, dnswire.NewMsg("example.org.", dnswire.TypeA)))
	require.NoError(t, err)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	require.NoError(t, server.listener.Close())
	select {
	case <-server.done:
	case <-time.After(time.Second):
		t.Fatal("listener failure did not stop active requests")
	}
}

func startTCPTest(t *testing.T, handler dnswire.Handler) (*TCP, *net.TCPConn) {
	t.Helper()
	return startTCPServer(t, handler, func(err error) { require.NoError(t, err) })
}

func startTCPServer(t *testing.T, handler dnswire.Handler, checkServe func(error)) (*TCP, *net.TCPConn) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := NewTCP(listener, handler)
	done := make(chan error, 1)
	go func() { done <- server.Serve(t.Context()) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		require.NoError(t, server.Shutdown(ctx))
		select {
		case err := <-done:
			checkServe(err)
		case <-ctx.Done():
			t.Error("TCP server did not stop")
		}
	})
	conn, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	return server, conn
}

func tcpTestFrame(t *testing.T, query *dnswire.Msg) []byte {
	t.Helper()
	require.NoError(t, query.Pack())
	return append(binary.BigEndian.AppendUint16(nil, uint16(len(query.Data))), query.Data...)
}

func readTCPTestReply(t *testing.T, conn net.Conn) *dnswire.Msg {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	reply := new(dnswire.Msg)
	reply.Data = make([]byte, dnswire.MinMsgSize)
	_, err := reply.ReadFrom(conn)
	require.NoError(t, err)
	require.NoError(t, reply.Unpack())
	require.True(t, reply.Response)
	return reply
}

func tcpTestReply(t *testing.T, w dnswire.ResponseWriter, request *dnswire.Msg) {
	t.Helper()
	reply := new(dnswire.Msg)
	dnsutil.SetReply(reply, request)
	if err := reply.Pack(); err != nil {
		t.Errorf("pack TCP reply: %v", err)
		return
	}
	if _, err := io.Copy(w, reply); err != nil {
		t.Errorf("write TCP reply: %v", err)
	}
}
