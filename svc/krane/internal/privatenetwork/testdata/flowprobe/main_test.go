package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func localEcho(t *testing.T) string {
	t.Helper()
	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	udp, err := net.ListenPacket("udp", tcp.Addr().String())
	require.NoError(t, err)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		if err := echoUDP(udp); !errors.Is(err, net.ErrClosed) {
			t.Errorf("UDP echo: %v", err)
		}
	}()
	go func() {
		defer workers.Done()
		for {
			conn, err := tcp.Accept()
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				t.Errorf("accept: %v", err)
				return
			}
			workers.Add(1)
			go func() {
				defer workers.Done()
				if err := echoTCP(conn); err != nil {
					t.Errorf("TCP echo: %v", err)
				}
				if err := conn.Close(); err != nil {
					t.Errorf("close TCP: %v", err)
				}
			}()
		}
	}()
	t.Cleanup(func() {
		require.NoError(t, tcp.Close())
		require.NoError(t, udp.Close())
		workers.Wait()
	})
	return tcp.Addr().String()
}

func TestClientProtocol(t *testing.T) {
	address := localEcho(t)
	var output bytes.Buffer
	require.NoError(t, client(strings.NewReader("fresh\nopen\nheld\nopen\nheld\nfresh\n"), &output, address))
	decoder := json.NewDecoder(&output)
	for range 6 {
		var got result
		require.NoError(t, decoder.Decode(&got))
		require.Equal(t, result{TCP: true, UDP: true}, got)
	}
	var extra result
	require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
	for _, command := range []string{"invalid", "held"} {
		t.Run(command, func(t *testing.T) {
			require.Error(t, client(strings.NewReader(command+"\n"), io.Discard, address))
		})
	}
}

func TestEchoExchanges(t *testing.T) {
	address := localEcho(t)
	for _, network := range []string{"tcp", "udp"} {
		t.Run(network, func(t *testing.T) {
			f, err := dial(network, address)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, f.conn.Close()) })
			for range 3 {
				ok, err := f.exchange()
				require.NoError(t, err)
				require.True(t, ok)
			}
		})
	}
}

func TestHeldTCPDiscardsBufferedLateReply(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { require.NoError(t, clientConn.Close()) })
	f := &flow{conn: clientConn, reader: bufio.NewReader(clientConn)}
	done := make(chan error, 1)
	go func() {
		defer close(done)
		defer func() {
			if err := serverConn.Close(); err != nil {
				t.Errorf("close server: %v", err)
			}
		}()
		reader := bufio.NewReader(serverConn)
		first, err := reader.ReadString('\n')
		if err != nil {
			done <- err
			return
		}
		if _, err := io.WriteString(serverConn, first[:8]); err != nil {
			done <- err
			return
		}
		second, err := reader.ReadString('\n')
		if err != nil {
			done <- err
			return
		}
		if first == second {
			done <- errors.New("exchange reused token")
			return
		}
		_, err = io.WriteString(serverConn, first[8:]+second)
		done <- err
	}()
	started := time.Now()
	ok, err := f.exchange()
	require.NoError(t, err)
	require.False(t, ok)
	require.GreaterOrEqual(t, time.Since(started), timeout)
	require.Less(t, time.Since(started), 2*time.Second)
	ok, err = f.exchange()
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, <-done)
}

func TestUDPDeadline(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	f, err := dial("udp", conn.LocalAddr().String())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.conn.Close()) })
	started := time.Now()
	ok, err := f.exchange()
	require.NoError(t, err)
	require.False(t, ok)
	require.GreaterOrEqual(t, time.Since(started), timeout)
	require.Less(t, time.Since(started), 2*time.Second)
}
