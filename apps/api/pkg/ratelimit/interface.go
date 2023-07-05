package ratelimit

type Ratelimiter interface {
	Take(req RatelimitRequest) RatelimitResponse
}

type RatelimitRequest struct {
	Identifier     string
	Max            int64
	RefillRate     int64
	RefillInterval int64
}

type RatelimitResponse struct {
	Pass      bool
	Limit     int64
	Remaining int64
	Reset     int64
}
