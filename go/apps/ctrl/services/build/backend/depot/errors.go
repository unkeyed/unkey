package depot

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
)

// These patterns indicate terminal errors that should NOT be retried
var terminalErrorPatterns = []string{
	"exit code:",                  // Usually terminal errors thrown by upstream providers contains that.
	"failed to solve",             // This happens when it fails to fetch an image or build from source
	"failed to compute cache key", // Broken dockerfile
	"no such file or directory",   // This happens when user passes wrong dokcerfile path
	"internal error",              // This is mostly thrown by depot. It means something is wrong on their end.
	"permission denied",           // This is on us we either pass wrong depot token or organization/project mismatch
	"unauthenticated",             // Wrong depot token
}

// isTerminalBuildError returns true if the error is terminal and Restate should NOT retry.
// In some cases such as broken dockerfile, internal error, wrong dockerfile path, there is no reason to keep retrying.
// We could retry, but those would pollute the CH logs
func isTerminalBuildError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	for _, pattern := range terminalErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

func wrapBuildError(err error, code connect.Code, msg string) error {
	wrapped := fmt.Errorf("%s: %w", msg, err)

	if isTerminalBuildError(err) {
		return restate.TerminalError(wrapped)
	}

	return connect.NewError(code, wrapped)
}
