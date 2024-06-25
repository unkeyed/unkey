package ratelimit

import (
	"sync"
	"time"
)

type identifier struct {
	sync.RWMutex
	identifier string
	windows    map[int64]*window
}

func (i *identifier) take(req RatelimitRequest) RatelimitResponse {
	i.Lock()
	defer i.Unlock()

	now := time.Now()
	duration := time.Duration(req.RefillInterval) * time.Millisecond
	windowId := now.UnixMilli() / int64(duration.Milliseconds())

	w, ok := i.windows[windowId]
	if !ok {
		w = &window{id: windowId, reset: now.Add(duration), count: 0}
		i.windows[windowId] = w
	}
	if w.count+req.Cost > req.Max {
		return RatelimitResponse{Pass: false, Limit: req.Max, Remaining: req.Max - w.count, Reset: w.reset.UnixMilli()}
	}
	w.count += req.Cost
	return RatelimitResponse{Pass: true, Limit: req.Max, Remaining: req.Max - w.count, Reset: w.reset.UnixMilli()}

}

type window struct {
	// unix milli timestamp of the start of the window
	id int64

	// we don't need an atomic here since this is only accessed while holding the identifier lock
	count int64

	reset time.Time
}

type fixedWindow struct {
	identifiersLock sync.RWMutex
	identifiers     map[string]*identifier
}

func NewFixedWindow() *fixedWindow {

	r := &fixedWindow{
		identifiersLock: sync.RWMutex{},
		identifiers:     make(map[string]*identifier),
	}

	go func() {
		for range time.NewTicker(time.Minute).C {
			now := time.Now()
			r.identifiersLock.Lock()

			for _, identifier := range r.identifiers {
				identifier.Lock()

				for _, w := range identifier.windows {
					if now.After(w.reset) {
						delete(identifier.windows, w.id)
					}
				}
				if len(identifier.windows) == 0 {

					delete(r.identifiers, identifier.identifier)
				}
				identifier.Unlock()
			}
			r.identifiersLock.Unlock()

		}
	}()

	return r

}

func (r *fixedWindow) Take(req RatelimitRequest) RatelimitResponse {

	r.identifiersLock.RLock()

	id, ok := r.identifiers[req.Identifier]
	r.identifiersLock.RUnlock()
	if ok {
		return id.take(req)
	}

	r.identifiersLock.Lock()
	// Check again since we are in a new lock and another goroutine could have created it now
	id, ok = r.identifiers[req.Identifier]
	if ok {
		r.identifiersLock.Unlock()
		return id.take(req)
	}

	id = &identifier{identifier: req.Identifier, windows: make(map[int64]*window)}
	r.identifiers[req.Identifier] = id
	r.identifiersLock.Unlock()

	return id.take(req)

}

func (r *fixedWindow) SetCurrent(req SetCurrentRequest) error {
	r.identifiersLock.Lock()
	id, ok := r.identifiers[req.Identifier]
	if !ok {
		id = &identifier{identifier: req.Identifier, windows: make(map[int64]*window)}
		r.identifiers[req.Identifier] = id
	}
	r.identifiersLock.Unlock()
	id.Lock()
	defer id.Unlock()

	now := time.Now()
	duration := time.Duration(req.RefillInterval) * time.Millisecond
	windowId := now.UnixMilli() / int64(duration.Milliseconds())

	w, ok := id.windows[windowId]
	if !ok {
		w = &window{id: windowId, reset: now.Add(duration), count: 0}
		id.windows[windowId] = w
	}

	w.count = req.Current

	return nil
}
