package port

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

type FreePort struct {
	sync.RWMutex
	min      int
	max      int
	attempts int
}

func New() *FreePort {
	rand.Seed(time.Now().UnixNano())
	return &FreePort{
		min:      50000,
		max:      65535,
		attempts: 10,
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

		ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
		if err != nil {
			continue
		}
		err = ln.Close()
		if err != nil {
			return -1, err
		}
		return port, nil
	}
	return -1, fmt.Errorf("could not find a free port, maybe increase attempts?")
}
