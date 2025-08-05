package codes

// userBadRequest defines errors related to invalid user input and bad requests.
type gatewayUnavailiable struct {
	// BadGateway indicates that the upstream server is unavailable.
	BadGateway Code
}

// GatewayErrors defines all gateway-related errors in the Unkey system.
// These errors are caused by invalid gateway inputs or client behavior.
type GatewayErrors struct {
	// BadRequest contains errors related to invalid gateway input.
	BadRequest gatewayUnavailiable
}

// Gateway contains all predefined gateway error codes.
// These errors can be referenced directly (e.g., codes.Gateway.BadRequest.QueryEmpty)
// for consistent error handling throughout the application.
var Gateway = GatewayErrors{
	BadRequest: gatewayUnavailiable{
		BadGateway: Code{SystemGateway, CategoryBadGateway, "gateway_unavailable"},
	},
}
