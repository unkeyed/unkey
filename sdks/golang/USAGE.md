<!-- Start SDK Example Usage [usage] -->
```go
package main

import (
	"context"
	"github.com/unkeyed/unkey/sdks/golang"
	"github.com/unkeyed/unkey/sdks/golang/models/components"
	"github.com/unkeyed/unkey/sdks/golang/models/operations"
	"log"
)

func main() {
	s := golang.New(
		golang.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
	)

	request := operations.CreateAPIRequestBody{
		Name: "my-api",
	}

	ctx := context.Background()
	res, err := s.CreateAPI(ctx, request)
	if err != nil {
		log.Fatal(err)
	}
	if res != nil {
		// handle response
	}
}

```
<!-- End SDK Example Usage [usage] -->