# Keys
(*Keys*)

### Available Operations

* [GetKey](#getkey)
* [DeleteKey](#deletekey)
* [CreateKey](#createkey)
* [VerifyKey](#verifykey)
* [UpdateKey](#updatekey)
* [UpdateRemaining](#updateremaining)
* [GetVerifications](#getverifications)

## GetKey

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/golang/models/components"
	"github.com/unkeyed/unkey/sdks/golang"
	"github.com/unkeyed/unkey/sdks/golang/models/operations"
	"context"
	"log"
)

func main() {
    s := golang.New(
        golang.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    request := operations.GetKeyRequest{
        KeyID: "key_1234",
    }
    
    ctx := context.Background()
    res, err := s.Keys.GetKey(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                            | Type                                                                 | Required                                                             | Description                                                          |
| -------------------------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `ctx`                                                                | [context.Context](https://pkg.go.dev/context#Context)                | :heavy_check_mark:                                                   | The context to use for the request.                                  |
| `request`                                                            | [operations.GetKeyRequest](../../models/operations/getkeyrequest.md) | :heavy_check_mark:                                                   | The request object to use for the request.                           |


### Response

**[*components.Key](../../models/components/key.md), error**
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

## DeleteKey

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/golang/models/components"
	"github.com/unkeyed/unkey/sdks/golang"
	"github.com/unkeyed/unkey/sdks/golang/models/operations"
	"context"
	"log"
)

func main() {
    s := golang.New(
        golang.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    request := operations.DeleteKeyRequestBody{
        KeyID: "key_1234",
    }
    
    ctx := context.Background()
    res, err := s.Keys.DeleteKey(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.DeleteKeyRequestBody](../../models/operations/deletekeyrequestbody.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |


### Response

**[*operations.DeleteKeyResponseBody](../../models/operations/deletekeyresponsebody.md), error**
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

## CreateKey

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/golang/models/components"
	"github.com/unkeyed/unkey/sdks/golang"
	"github.com/unkeyed/unkey/sdks/golang/models/operations"
	"context"
	"log"
)

func main() {
    s := golang.New(
        golang.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    request := operations.CreateKeyRequestBody{
        APIID: "api_123",
        Name: golang.String("my key"),
        OwnerID: golang.String("team_123"),
        Meta: map[string]interface{}{
            "billingTier": "PRO",
            "trialEnds": "2023-06-16T17:16:37.161Z",
        },
        Roles: []string{
            "admin",
            "finance",
        },
        Expires: golang.Int64(1623869797161),
        Remaining: golang.Int64(1000),
        Refill: &operations.Refill{
            Interval: operations.IntervalDaily,
            Amount: 100,
        },
        Ratelimit: &operations.Ratelimit{
            Type: operations.TypeFast.ToPointer(),
            Limit: 10,
            RefillRate: 1,
            RefillInterval: 60,
        },
        Enabled: golang.Bool(false),
    }
    
    ctx := context.Background()
    res, err := s.Keys.CreateKey(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.CreateKeyRequestBody](../../models/operations/createkeyrequestbody.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |


### Response

**[*operations.CreateKeyResponseBody](../../models/operations/createkeyresponsebody.md), error**
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

## VerifyKey

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/golang/models/components"
	"github.com/unkeyed/unkey/sdks/golang"
	"context"
	"log"
)

func main() {
    s := golang.New(
        golang.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    request := components.V1KeysVerifyKeyRequest{
        APIID: golang.String("api_1234"),
        Key: "sk_1234",
        Authorization: &components.Authorization{
            Permissions: &components.Permissions{},
        },
    }
    
    ctx := context.Background()
    res, err := s.Keys.VerifyKey(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                              | Type                                                                                   | Required                                                                               | Description                                                                            |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `ctx`                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                  | :heavy_check_mark:                                                                     | The context to use for the request.                                                    |
| `request`                                                                              | [components.V1KeysVerifyKeyRequest](../../models/components/v1keysverifykeyrequest.md) | :heavy_check_mark:                                                                     | The request object to use for the request.                                             |


### Response

**[*components.V1KeysVerifyKeyResponse](../../models/components/v1keysverifykeyresponse.md), error**
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

## UpdateKey

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/golang/models/components"
	"github.com/unkeyed/unkey/sdks/golang"
	"github.com/unkeyed/unkey/sdks/golang/models/operations"
	"context"
	"log"
)

func main() {
    s := golang.New(
        golang.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    request := operations.UpdateKeyRequestBody{
        KeyID: "key_123",
        Name: golang.String("Customer X"),
        OwnerID: golang.String("user_123"),
        Meta: map[string]interface{}{
            "roles": "<value>",
            "stripeCustomerId": "cus_1234",
        },
        Expires: golang.Float64(0),
        Ratelimit: &operations.UpdateKeyRatelimit{
            Type: operations.UpdateKeyTypeFast,
            Limit: 10,
            RefillRate: 1,
            RefillInterval: 60,
        },
        Remaining: golang.Float64(1000),
        Refill: &operations.UpdateKeyRefill{
            Interval: operations.UpdateKeyIntervalDaily,
            Amount: 100,
        },
        Enabled: golang.Bool(true),
    }
    
    ctx := context.Background()
    res, err := s.Keys.UpdateKey(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.UpdateKeyRequestBody](../../models/operations/updatekeyrequestbody.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |


### Response

**[*operations.UpdateKeyResponseBody](../../models/operations/updatekeyresponsebody.md), error**
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

## UpdateRemaining

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/golang/models/components"
	"github.com/unkeyed/unkey/sdks/golang"
	"github.com/unkeyed/unkey/sdks/golang/models/operations"
	"context"
	"log"
)

func main() {
    s := golang.New(
        golang.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    request := operations.UpdateRemainingRequestBody{
        KeyID: "key_123",
        Op: operations.OpSet,
        Value: golang.Int64(1),
    }
    
    ctx := context.Background()
    res, err := s.Keys.UpdateRemaining(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                      | Type                                                                                           | Required                                                                                       | Description                                                                                    |
| ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `ctx`                                                                                          | [context.Context](https://pkg.go.dev/context#Context)                                          | :heavy_check_mark:                                                                             | The context to use for the request.                                                            |
| `request`                                                                                      | [operations.UpdateRemainingRequestBody](../../models/operations/updateremainingrequestbody.md) | :heavy_check_mark:                                                                             | The request object to use for the request.                                                     |


### Response

**[*operations.UpdateRemainingResponseBody](../../models/operations/updateremainingresponsebody.md), error**
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

## GetVerifications

### Example Usage

```go
package main

import(
	"github.com/unkeyed/unkey/sdks/golang/models/components"
	"github.com/unkeyed/unkey/sdks/golang"
	"github.com/unkeyed/unkey/sdks/golang/models/operations"
	"context"
	"log"
)

func main() {
    s := golang.New(
        golang.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    request := operations.GetVerificationsRequest{
        KeyID: golang.String("key_1234"),
        OwnerID: golang.String("chronark"),
        Start: golang.Int64(1620000000000),
        End: golang.Int64(1620000000000),
        Granularity: operations.GranularityDay.ToPointer(),
    }
    
    ctx := context.Background()
    res, err := s.Keys.GetVerifications(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `request`                                                                                | [operations.GetVerificationsRequest](../../models/operations/getverificationsrequest.md) | :heavy_check_mark:                                                                       | The request object to use for the request.                                               |


### Response

**[*operations.GetVerificationsResponseBody](../../models/operations/getverificationsresponsebody.md), error**
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
