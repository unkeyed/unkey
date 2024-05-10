# Apis
(*Apis*)

### Available Operations

* [V1ApisGetAPI](#v1apisgetapi)
* [V1ApisListKeys](#v1apislistkeys)

## V1ApisGetAPI

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/goclient/models/components"
	"github.com/unkeyed/unkey/sdks/goclient"
	"context"
	"log"
)

func main() {
    s := goclient.New(
        goclient.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    var apiID string = "api_1234"
    
    ctx := context.Background()
    res, err := s.Apis.V1ApisGetAPI(ctx, apiID)
    if err != nil {
        log.Fatal(err)
    }
    if res.Object != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                             | Type                                                  | Required                                              | Description                                           | Example                                               |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `ctx`                                                 | [context.Context](https://pkg.go.dev/context#Context) | :heavy_check_mark:                                    | The context to use for the request.                   |                                                       |
| `apiID`                                               | *string*                                              | :heavy_check_mark:                                    | N/A                                                   | api_1234                                              |


### Response

**[*operations.V1ApisGetAPIResponse](../../models/operations/v1apisgetapiresponse.md), error**
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

## V1ApisListKeys

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/goclient/models/components"
	"github.com/unkeyed/unkey/sdks/goclient"
	"context"
	"log"
)

func main() {
    s := goclient.New(
        goclient.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    var apiID string = "api_1234"

    var limit *int64 = goclient.Int64(100)

    var cursor *string = goclient.String("<value>")

    var ownerID *string = goclient.String("<value>")
    
    ctx := context.Background()
    res, err := s.Apis.V1ApisListKeys(ctx, apiID, limit, cursor, ownerID)
    if err != nil {
        log.Fatal(err)
    }
    if res.Object != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                             | Type                                                  | Required                                              | Description                                           | Example                                               |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `ctx`                                                 | [context.Context](https://pkg.go.dev/context#Context) | :heavy_check_mark:                                    | The context to use for the request.                   |                                                       |
| `apiID`                                               | *string*                                              | :heavy_check_mark:                                    | N/A                                                   | api_1234                                              |
| `limit`                                               | **int64*                                              | :heavy_minus_sign:                                    | N/A                                                   | 100                                                   |
| `cursor`                                              | **string*                                             | :heavy_minus_sign:                                    | N/A                                                   |                                                       |
| `ownerID`                                             | **string*                                             | :heavy_minus_sign:                                    | N/A                                                   |                                                       |


### Response

**[*operations.V1ApisListKeysResponse](../../models/operations/v1apislistkeysresponse.md), error**
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
