package ratelimit

type Ratelimiter interface {
	Take(req RatelimitRequest) RatelimitResponse
}

type RatelimitRequest struct {
	Identifier     string
	Max            int32
	RefillRate     int32
	RefillInterval int32
}

type RatelimitResponse struct {
	Pass      bool
	Limit     int32
	Remaining int32
	Reset     int64
}
