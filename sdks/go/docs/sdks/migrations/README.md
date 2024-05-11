# Migrations
(*Migrations*)

### Available Operations

* [V1MigrationsCreateKeys](#v1migrationscreatekeys)

## V1MigrationsCreateKeys

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

    request := []operations.RequestBody{
        operations.RequestBody{
            APIID: "api_123",
            Name: goclient.String("my key"),
            Hash: operations.Hash{
                Value: "<value>",
                Variant: operations.VariantSha256Base64,
            },
            Start: goclient.String("unkey_32kq"),
            OwnerID: goclient.String("team_123"),
            Meta: map[string]interface{}{
                "billingTier": "PRO",
                "trialEnds": "2023-06-16T17:16:37.161Z",
            },
            Roles: []string{
                "admin",
                "finance",
            },
            Expires: goclient.Int64(1623869797161),
            Remaining: goclient.Int64(1000),
            Refill: &operations.V1MigrationsCreateKeysRefill{
                Interval: operations.V1MigrationsCreateKeysIntervalDaily,
                Amount: 100,
            },
            Ratelimit: &operations.V1MigrationsCreateKeysRatelimit{
                Type: operations.V1MigrationsCreateKeysTypeFast.ToPointer(),
                Limit: 10,
                RefillRate: 1,
                RefillInterval: 60,
            },
            Enabled: goclient.Bool(false),
        },
    }
    
    ctx := context.Background()
    res, err := s.Migrations.V1MigrationsCreateKeys(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                             | Type                                                  | Required                                              | Description                                           |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `ctx`                                                 | [context.Context](https://pkg.go.dev/context#Context) | :heavy_check_mark:                                    | The context to use for the request.                   |
| `request`                                             | [[]operations.RequestBody](../../.md)                 | :heavy_check_mark:                                    | The request object to use for the request.            |


### Response

**[*operations.V1MigrationsCreateKeysResponseBody](../../models/operations/v1migrationscreatekeysresponsebody.md), error**
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
