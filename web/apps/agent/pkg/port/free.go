package port

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// FreePort is a utility to find a free port.
type FreePort struct {
	sync.RWMutex
	min      int
	max      int
	attempts int

	// The caller may request multiple ports without binding them immediately
	// so we need to keep track of which ports are assigned.
	assigned map[int]bool
}

func New() *FreePort {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	return &FreePort{
		min:      10000,
		max:      65535,
		attempts: 10,
		assigned: map[int]bool{},
	}
}
func (f *FreePort) Get() int {
	port, err := f.GetWithError()
	if err != nil {
		panic(err)
	}

	return port
}

// Get returns a free port.
func (f *FreePort) GetWithError() (int, error) {
	f.Lock()
	defer f.Unlock()

	for i := 0; i < f.attempts; i++ {

		port := rand.Intn(f.max-f.min) + f.min
		if f.assigned[port] {
			continue
		}

		ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
		if err != nil {
			continue
		}
		err = ln.Close()
		if err != nil {
			return -1, err
		}
		f.assigned[port] = true
		return port, nil
	}
	return -1, fmt.Errorf("could not find a free port, maybe increase attempts?")
}
