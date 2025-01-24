package systemerrors

import "fmt"

type Fault string

const (
	AWS    Fault = "AWS"
	Unkey  Fault = "Unkey"
	GitHub Fault = "GitHub"
)

type Service string

const (
	AppRunner   Service = "AppRunner"
	Route53     Service = "Route53"
	UnkeyDeploy Service = "UnkeyDeploy"
)

type Code string

const (
	ACCESS_DENIED Code = "ACCESS_DENIED"
)

// EID - Error ID
//
// Error ids are globally unique identifier for errors
// They consist of a fault, a service and a code and are created like so:
// "EID:{Fault}:{Service}:{Code}"
//
// For example "EID:AWS:Route53:ACCESS_DENIED".
type EID string

type Error struct {
	Fault   Fault
	Service Service
	Code    Code
}

func (e Error) EID() EID {
	return EID(fmt.Sprintf("EID:%s:%s:%s", e.Fault, e.Service, e.Code))
}
