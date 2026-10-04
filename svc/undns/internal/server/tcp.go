package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/unkeyed/unkey/pkg/logger"
)

// TCP serves pipelined DNS requests over TCP. Replies to one connection are
// written whole, each within a 2 second deadline, and a connection closes only
// after every reply to it has been written. It closes after
// dnswire.MaxTCPQueries requests or an idle read timeout.
type TCP struct {
	listener net.Listener
	handler  dnswire.Handler
	done     chan struct{}
	mu       sync.Mutex
	closing  bool
	peers    map[*tcpResponse]tcpCancellation
}

type tcpCancellation struct {
	read     context.CancelFunc
	requests context.CancelFunc
}

// NewTCP returns a server that takes ownership of listener and answers with
// handler.
func NewTCP(listener net.Listener, handler dnswire.Handler) *TCP {
	return &TCP{
		listener: listener,
		handler:  handler,
		done:     make(chan struct{}),
		mu:       sync.Mutex{},
		closing:  false,
		peers:    make(map[*tcpResponse]tcpCancellation),
	}
}

// Serve accepts connections until the listener stops and returns after every
// accepted request has finished. It returns nil after [TCP.Shutdown] and the
// listener error, after cancelling active requests, if the listener fails.
// Call it once; cancellation of ctx does not stop accepted requests.
func (s *TCP) Serve(ctx context.Context) error {
	defer close(s.done)
	var peers sync.WaitGroup
	defer peers.Wait()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			closing := s.closing
			if !closing {
				for peer, cancel := range s.peers {
					cancel.requests()
					err = errors.Join(err, peer.Close())
				}
			}
			s.mu.Unlock()
			if closing {
				return nil
			}
			return fmt.Errorf("accept DNS TCP connection: %w", err)
		}

		w := &tcpResponse{conn: conn, writeMu: sync.Mutex{}}
		s.mu.Lock()
		if s.closing {
			s.mu.Unlock()
			return w.Close()
		}
		requests, cancelRequests := context.WithCancel(context.WithoutCancel(ctx))
		reading, cancelRead := context.WithCancel(requests)
		s.peers[w] = tcpCancellation{read: cancelRead, requests: cancelRequests}
		s.mu.Unlock()
		peers.Go(func() {
			s.servePeer(requests, reading, w)
			cancelRead()
			cancelRequests()
			if err := w.Close(); err != nil {
				logger.Warn("close DNS TCP connection", "error", err)
			}
			s.mu.Lock()
			delete(s.peers, w)
			s.mu.Unlock()
		})
	}
}

// Shutdown stops accepting and reading, draining accepted requests until ctx
// expires, then cancelling requests and closing their connections.
func (s *TCP) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.closing = true
	err := s.listener.Close()
	for _, cancel := range s.peers {
		cancel.read()
	}
	s.mu.Unlock()
	if errors.Is(err, net.ErrClosed) {
		err = nil
	}

	select {
	case <-s.done:
		return err
	case <-ctx.Done():
		s.mu.Lock()
		for peer, cancel := range s.peers {
			cancel.requests()
			err = errors.Join(err, peer.Close())
		}
		s.mu.Unlock()
		return errors.Join(err, ctx.Err())
	}
}

func (s *TCP) servePeer(requests, reading context.Context, w *tcpResponse) {
	stop := context.AfterFunc(reading, func() {
		if err := w.conn.SetReadDeadline(time.Now()); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Warn("interrupt DNS TCP read", "error", err)
		}
	})
	defer stop()
	var replies sync.WaitGroup
	defer replies.Wait()

	for query := range dnswire.MaxTCPQueries {
		timeout := 8 * time.Second
		if query == 0 {
			timeout = 5 * time.Second
		}
		if err := w.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil || reading.Err() != nil {
			return
		}

		request := new(dnswire.Msg)
		request.Data = make([]byte, dnswire.MinMsgSize)
		if _, err := request.ReadFrom(w.conn); err != nil {
			return
		}
		request.Options = dnswire.MsgOptionUnpackQuestion
		if err := request.Unpack(); err != nil {
			continue
		}
		replies.Go(func() { s.answer(requests, w, request) })
	}
}

func (s *TCP) answer(ctx context.Context, w *tcpResponse, request *dnswire.Msg) {
	var code uint16
	switch dnswire.DefaultMsgAcceptFunc(request) {
	case dnswire.MsgIgnore:
		return
	case dnswire.MsgAccept:
		request.Options = dnswire.MsgOptionUnpack
		s.handler.ServeDNS(ctx, w, request)
		return
	case dnswire.MsgReject:
		code = dnswire.RcodeFormatError
	case dnswire.MsgRejectNotImplemented:
		code = dnswire.RcodeNotImplemented
	case dnswire.MsgRejectRefused:
		code = dnswire.RcodeRefused
	}

	response := new(dnswire.Msg)
	dnsutil.SetReply(response, request)
	response.Rcode = code
	if _, err := io.Copy(w, response); err != nil {
		logger.Warn("write rejected DNS TCP response", "error", err)
	}
}

type tcpResponse struct {
	conn    net.Conn
	writeMu sync.Mutex
}

func (w *tcpResponse) LocalAddr() net.Addr       { return w.conn.LocalAddr() }
func (w *tcpResponse) RemoteAddr() net.Addr      { return w.conn.RemoteAddr() }
func (w *tcpResponse) Conn() net.Conn            { return w.conn }
func (w *tcpResponse) Session() *dnswire.Session { return nil }
func (w *tcpResponse) Hijack()                   { panic("undns does not support TCP hijacking") }

func (w *tcpResponse) Close() error {
	err := w.conn.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (w *tcpResponse) Write(p []byte) (int, error) {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return 0, errors.Join(err, w.Close())
	}

	n, err := w.conn.Write(p)
	if n != len(p) && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		err = errors.Join(err, w.Close())
	}
	return n, err
}
