package proxy

const copyBufferSize = 32 << 10

var responseCopyBuffers = make(copyBufferPool, 64)

type copyBufferPool chan []byte

func (p copyBufferPool) Get() []byte {
	select {
	case b := <-p:
		return b
	default:
		return make([]byte, copyBufferSize)
	}
}

func (p copyBufferPool) Put(b []byte) {
	clear(b)
	select {
	case p <- b:
	default:
	}
}
