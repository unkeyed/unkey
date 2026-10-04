package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

const timeout = 500 * time.Millisecond

type result struct {
	TCP bool `json:"tcp"`
	UDP bool `json:"udp"`
}

type flow struct {
	conn    net.Conn
	reader  *bufio.Reader
	pending string
}

func dial(network, address string) (*flow, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	return &flow{conn: conn, reader: bufio.NewReader(conn), pending: ""}, nil
}

func (f *flow) exchange() (bool, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return false, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(nonce[:]) + "\n"
	if err := f.conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return false, fmt.Errorf("set deadline: %w", err)
	}
	if _, err := io.WriteString(f.conn, token); err != nil {
		return false, nil
	}
	for {
		line, err := f.reader.ReadString('\n')
		f.pending += line
		if strings.HasSuffix(f.pending, "\n") {
			matched := f.pending == token
			f.pending = ""
			if matched {
				return true, nil
			}
		}
		if err != nil {
			return false, nil
		}
	}
}

func client(input io.Reader, output io.Writer, address string) (err error) {
	held := make(map[string]*flow)
	defer func() {
		for _, f := range held {
			if closeErr := f.conn.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}
	}()
	scanner := bufio.NewScanner(input)
	encoder := json.NewEncoder(output)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if command != "fresh" && command != "open" && command != "held" {
			return fmt.Errorf("unknown command %q", command)
		}
		var response result
		for _, network := range []string{"tcp", "udp"} {
			f := held[network]
			if command != "held" {
				var dialErr error
				f, dialErr = dial(network, address)
				if dialErr != nil {
					continue
				}
			}
			if f == nil {
				return fmt.Errorf("no held %s flow", network)
			}
			ok, err := f.exchange()
			if err != nil {
				if command != "held" {
					err = errors.Join(err, f.conn.Close())
				}
				return err
			}
			if network == "tcp" {
				response.TCP = ok
			} else {
				response.UDP = ok
			}
			if command == "held" {
				continue
			}
			if command == "open" && ok {
				old := held[network]
				held[network] = f
				if old != nil {
					if err := old.conn.Close(); err != nil {
						return err
					}
				}
			} else if err := f.conn.Close(); err != nil {
				return err
			}
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func echoTCP(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if _, err := io.WriteString(conn, line); err != nil {
			return err
		}
	}
}

func echoUDP(conn net.PacketConn) error {
	buffer := make([]byte, 4096)
	for {
		n, address, err := conn.ReadFrom(buffer)
		if err != nil {
			return err
		}
		if _, err := conn.WriteTo(buffer[:n], address); err != nil {
			return err
		}
	}
}

func server(address string) (err error) {
	tcp, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, tcp.Close()) }()
	udp, err := net.ListenPacket("udp", address)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, udp.Close()) }()
	go func() {
		if err := echoUDP(udp); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()
	for {
		conn, err := tcp.Accept()
		if err != nil {
			return err
		}
		go func() {
			if err := echoTCP(conn); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			if err := conn.Close(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()
	}
}

func main() {
	var err error
	switch {
	case len(os.Args) == 2 && os.Args[1] == "server":
		err = server(":8080")
	case len(os.Args) == 3 && os.Args[1] == "client":
		err = client(os.Stdin, os.Stdout, net.JoinHostPort(os.Args[2], "8080"))
	default:
		err = fmt.Errorf("usage: flowprobe server | client HOST")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
