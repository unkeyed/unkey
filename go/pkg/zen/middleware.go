package zen

type Middleware func(handler HandleFunc) HandleFunc
