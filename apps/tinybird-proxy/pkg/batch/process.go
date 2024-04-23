package batch

import (
	"context"
	"time"
)




type BatchProcessor[T any] struct {
	buffer chan T
	batch []T
	flush  func(ctx context.Context, batch []T) 
	ticker *time.Ticker


}


type Config[T any] struct {
	BatchSize int64
	BufferSize int64
	FlushInterval time.Duration
	Flush func(ctx context.Context, batch []T) 
}

func New[T any](config Config[T]) *BatchProcessor[T] {
	bp := &BatchProcessor[T]{
		buffer: make(chan T, config.BufferSize),
		batch: make([]T, 0,config.BatchSize),
		flush:  config.Flush,
		ticker: time.NewTicker(config.FlushInterval),
	}


		flushAndReset := func(){
			if len(bp.batch) > 0 {
				bp.flush(context.Background(), bp.batch)
				bp.batch = bp.batch[:0]
			}
			bp.ticker.Reset(config.FlushInterval)
		}
	
		go func() {
			for {
				select {
				case e, ok := <-bp.buffer:
					if !ok {
						// channel closed
						if len(bp.batch) > 0 {
							config.Flush(context.Background(), bp.batch)
							bp.batch = bp.batch[:0]
						}
						bp.ticker.Stop()
						return
					}
					bp.batch = append(bp.batch, e)
					if len(bp.batch) >= int(config.BatchSize) {
					flushAndReset()
	
					}
				case <-bp.ticker.C:
				flushAndReset()
				}
			}
		}()


	
	return bp
}



func (bp *BatchProcessor[T]) Buffer(t T)  {
	bp.buffer <- t
}

func (bp *BatchProcessor[T]) Close() {
	close(bp.buffer)
}
