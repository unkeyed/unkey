<!-- Start SDK Example Usage [usage] -->
```go
package main

import (
	"context"
	"github.com/unkeyed/unkey/sdks/goclient"
	"github.com/unkeyed/unkey/sdks/goclient/models/components"
	"github.com/unkeyed/unkey/sdks/goclient/models/operations"
	"log"
)

func main() {
	s := goclient.New(
		goclient.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
	)

	request := operations.V1ApisCreateAPIRequestBody{
		Name: "my-api",
	}

	ctx := context.Background()
	res, err := s.V1ApisCreateAPI(ctx, request)
	if err != nil {
		log.Fatal(err)
	}
	if res != nil {
		// handle response
	}
}

```
<!-- End SDK Example Usage [usage] -->