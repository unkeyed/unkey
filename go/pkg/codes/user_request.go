package codes

// userBadRequest defines errors related to invalid user input and bad requests.
type userBadRequest struct {
	// PermissionsQuerySyntaxError indicates a syntax or lexical error in verifyKey permissions query parsing.
	PermissionsQuerySyntaxError Code
	// RequestBodyTooLarge indicates the request body exceeds the maximum allowed size.
	RequestBodyTooLarge Code
	// RequestTimeout indicates the request took too long to process.
	RequestTimeout Code
	// ClientClosedRequest indicates the client closed the connection before the request completed.
	ClientClosedRequest Code
}

// UserErrors defines all user-related errors in the Unkey system.
// These errors are caused by invalid user inputs or client behavior.
type UserErrors struct {
	// BadRequest contains errors related to invalid user input.
	BadRequest userBadRequest
}

// User contains all predefined user error codes.
// These errors can be referenced directly (e.g., codes.User.BadRequest.QueryEmpty)
// for consistent error handling throughout the application.
var User = UserErrors{
	BadRequest: userBadRequest{
		PermissionsQuerySyntaxError: Code{SystemUser, CategoryUserBadRequest, "permissions_query_syntax_error"},
		RequestBodyTooLarge:         Code{SystemUser, CategoryUserBadRequest, "request_body_too_large"},
		RequestTimeout:              Code{SystemUser, CategoryUserBadRequest, "request_timeout"},
		ClientClosedRequest:         Code{SystemUser, CategoryUserBadRequest, "client_closed_request"},
	},
}
