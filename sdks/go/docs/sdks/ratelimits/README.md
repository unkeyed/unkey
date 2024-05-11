# Ratelimits
(*Ratelimits*)

### Available Operations

* [V1RatelimitsLimit](#v1ratelimitslimit)

## V1RatelimitsLimit

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/goclient/models/components"
	"github.com/unkeyed/unkey/sdks/goclient"
	"github.com/unkeyed/unkey/sdks/goclient/models/operations"
	"context"
	"log"
)

func main() {
    s := goclient.New(
        goclient.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    request := operations.V1RatelimitsLimitRequestBody{
        Namespace: goclient.String("email.outbound"),
        Identifier: "user_123",
        Limit: 10,
        Duration: 60000,
        Cost: goclient.Int64(2),
        Resources: []operations.Resources{
            operations.Resources{
                Type: "project",
                ID: "p_123",
                Name: goclient.String("dub"),
            },
        },
    }
    
    ctx := context.Background()
    res, err := s.Ratelimits.V1RatelimitsLimit(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                          | Type                                                                                               | Required                                                                                           | Description                                                                                        |
| -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                              | [context.Context](https://pkg.go.dev/context#Context)                                              | :heavy_check_mark:                                                                                 | The context to use for the request.                                                                |
| `request`                                                                                          | [operations.V1RatelimitsLimitRequestBody](../../models/operations/v1ratelimitslimitrequestbody.md) | :heavy_check_mark:                                                                                 | The request object to use for the request.                                                         |


### Response

**[*operations.V1RatelimitsLimitResponseBody](../../models/operations/v1ratelimitslimitresponsebody.md), error**
| Error Object                     | Status Code                      | Content Type                     |
| -------------------------------- | -------------------------------- | -------------------------------- |
| sdkerrors.ErrBadRequest          | 400                              | application/json                 |
| sdkerrors.ErrUnauthorized        | 401                              | application/json                 |
| sdkerrors.ErrForbidden           | 403                              | application/json                 |
| sdkerrors.ErrNotFound            | 404                              | application/json                 |
| sdkerrors.ErrConflict            | 409                              | application/json                 |
| sdkerrors.ErrTooManyRequests     | 429                              | application/json                 |
| sdkerrors.ErrInternalServerError | 500                              | application/json                 |
| sdkerrors.SDKError               | 4xx-5xx                          | */*                              |
