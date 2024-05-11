# Unkey SDK


## Overview

### Available Operations

* [CreateAPI](#createapi)
* [DeleteAPI](#deleteapi)
* [DeprecatedCreateKey](#deprecatedcreatekey)
* [~~DeprecatedVerifyKey~~](#deprecatedverifykey) - :warning: **Deprecated**
* [DeprecatedListKeys](#deprecatedlistkeys)

## CreateAPI

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

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.CreateAPIRequestBody](../../models/operations/createapirequestbody.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |


### Response

**[*operations.CreateAPIResponseBody](../../models/operations/createapiresponsebody.md), error**
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

## DeleteAPI

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

    request := operations.DeleteAPIRequestBody{
        APIID: "api_1234",
    }
    
    ctx := context.Background()
    res, err := s.DeleteAPI(ctx, request)
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
| `request`                                                                          | [operations.DeleteAPIRequestBody](../../models/operations/deleteapirequestbody.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |


### Response

**[*operations.DeleteAPIResponseBody](../../models/operations/deleteapiresponsebody.md), error**
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

## DeprecatedCreateKey

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

    request := operations.DeprecatedCreateKeyRequestBody{
        APIID: "api_123",
        Name: golang.String("my key"),
        OwnerID: golang.String("team_123"),
        Meta: map[string]interface{}{
            "billingTier": "PRO",
            "trialEnds": "2023-06-16T17:16:37.161Z",
        },
        Expires: golang.Int64(1623869797161),
        Remaining: golang.Int64(1000),
        Ratelimit: &operations.DeprecatedCreateKeyRatelimit{
            Type: operations.DeprecatedCreateKeyTypeFast.ToPointer(),
            Limit: 10,
            RefillRate: 1,
            RefillInterval: 60,
        },
    }
    
    ctx := context.Background()
    res, err := s.DeprecatedCreateKey(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                              | Type                                                                                                   | Required                                                                                               | Description                                                                                            |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                                  | :heavy_check_mark:                                                                                     | The context to use for the request.                                                                    |
| `request`                                                                                              | [operations.DeprecatedCreateKeyRequestBody](../../models/operations/deprecatedcreatekeyrequestbody.md) | :heavy_check_mark:                                                                                     | The request object to use for the request.                                                             |


### Response

**[*operations.DeprecatedCreateKeyResponseBody](../../models/operations/deprecatedcreatekeyresponsebody.md), error**
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

## ~~DeprecatedVerifyKey~~

> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

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

    request := operations.DeprecatedVerifyKeyRequestBody{
        APIID: golang.String("api_1234"),
        Key: "sk_1234",
    }
    
    ctx := context.Background()
    res, err := s.DeprecatedVerifyKey(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                              | Type                                                                                                   | Required                                                                                               | Description                                                                                            |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                                  | :heavy_check_mark:                                                                                     | The context to use for the request.                                                                    |
| `request`                                                                                              | [operations.DeprecatedVerifyKeyRequestBody](../../models/operations/deprecatedverifykeyrequestbody.md) | :heavy_check_mark:                                                                                     | The request object to use for the request.                                                             |


### Response

**[*operations.DeprecatedVerifyKeyResponseBody](../../models/operations/deprecatedverifykeyresponsebody.md), error**
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

## DeprecatedListKeys

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

    request := operations.DeprecatedListKeysRequest{
        APIID: "api_1234",
        Limit: golang.Int64(100),
    }
    
    ctx := context.Background()
    res, err := s.DeprecatedListKeys(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                    | Type                                                                                         | Required                                                                                     | Description                                                                                  |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `ctx`                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                        | :heavy_check_mark:                                                                           | The context to use for the request.                                                          |
| `request`                                                                                    | [operations.DeprecatedListKeysRequest](../../models/operations/deprecatedlistkeysrequest.md) | :heavy_check_mark:                                                                           | The request object to use for the request.                                                   |


### Response

**[*operations.DeprecatedListKeysResponseBody](../../models/operations/deprecatedlistkeysresponsebody.md), error**
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
