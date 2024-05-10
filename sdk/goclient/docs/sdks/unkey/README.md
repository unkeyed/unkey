# Unkey SDK


## Overview

### Available Operations

* [V1ApisCreateAPI](#v1apiscreateapi)
* [V1ApsisDeleteAPI](#v1apsisdeleteapi)
* [PostV1Keys](#postv1keys)
* [PostV1KeysVerify](#postv1keysverify)
* [GetV1ApisAPIIDKeys](#getv1apisapiidkeys)

## V1ApisCreateAPI

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

    request := operations.V1ApisCreateAPIRequestBody{
        Name: "my-api",
    }
    
    ctx := context.Background()
    res, err := s.V1ApisCreateAPI(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.Object != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                      | Type                                                                                           | Required                                                                                       | Description                                                                                    |
| ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `ctx`                                                                                          | [context.Context](https://pkg.go.dev/context#Context)                                          | :heavy_check_mark:                                                                             | The context to use for the request.                                                            |
| `request`                                                                                      | [operations.V1ApisCreateAPIRequestBody](../../models/operations/v1apiscreateapirequestbody.md) | :heavy_check_mark:                                                                             | The request object to use for the request.                                                     |


### Response

**[*operations.V1ApisCreateAPIResponse](../../models/operations/v1apiscreateapiresponse.md), error**
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

## V1ApsisDeleteAPI

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

    request := operations.V1ApsisDeleteAPIRequestBody{
        APIID: "api_1234",
    }
    
    ctx := context.Background()
    res, err := s.V1ApsisDeleteAPI(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.Object != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                        | Type                                                                                             | Required                                                                                         | Description                                                                                      |
| ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                            | [context.Context](https://pkg.go.dev/context#Context)                                            | :heavy_check_mark:                                                                               | The context to use for the request.                                                              |
| `request`                                                                                        | [operations.V1ApsisDeleteAPIRequestBody](../../models/operations/v1apsisdeleteapirequestbody.md) | :heavy_check_mark:                                                                               | The request object to use for the request.                                                       |


### Response

**[*operations.V1ApsisDeleteAPIResponse](../../models/operations/v1apsisdeleteapiresponse.md), error**
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

## PostV1Keys

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

    request := operations.PostV1KeysRequestBody{
        APIID: "api_123",
        Name: goclient.String("my key"),
        OwnerID: goclient.String("team_123"),
        Meta: map[string]interface{}{
            "billingTier": "PRO",
            "trialEnds": "2023-06-16T17:16:37.161Z",
        },
        Expires: goclient.Int64(1623869797161),
        Remaining: goclient.Int64(1000),
        Ratelimit: &operations.PostV1KeysRatelimit{
            Type: operations.PostV1KeysTypeFast.ToPointer(),
            Limit: 10,
            RefillRate: 1,
            RefillInterval: 60,
        },
    }
    
    ctx := context.Background()
    res, err := s.PostV1Keys(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.Object != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                            | Type                                                                                 | Required                                                                             | Description                                                                          |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| `ctx`                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                | :heavy_check_mark:                                                                   | The context to use for the request.                                                  |
| `request`                                                                            | [operations.PostV1KeysRequestBody](../../models/operations/postv1keysrequestbody.md) | :heavy_check_mark:                                                                   | The request object to use for the request.                                           |


### Response

**[*operations.PostV1KeysResponse](../../models/operations/postv1keysresponse.md), error**
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

## PostV1KeysVerify

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

    request := operations.PostV1KeysVerifyRequestBody{
        APIID: goclient.String("api_1234"),
        Key: "sk_1234",
    }
    
    ctx := context.Background()
    res, err := s.PostV1KeysVerify(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.Object != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                        | Type                                                                                             | Required                                                                                         | Description                                                                                      |
| ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                            | [context.Context](https://pkg.go.dev/context#Context)                                            | :heavy_check_mark:                                                                               | The context to use for the request.                                                              |
| `request`                                                                                        | [operations.PostV1KeysVerifyRequestBody](../../models/operations/postv1keysverifyrequestbody.md) | :heavy_check_mark:                                                                               | The request object to use for the request.                                                       |


### Response

**[*operations.PostV1KeysVerifyResponse](../../models/operations/postv1keysverifyresponse.md), error**
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

## GetV1ApisAPIIDKeys

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

    var offset *float64 = goclient.Float64(2757.36)

    var ownerID *string = goclient.String("<value>")
    
    ctx := context.Background()
    res, err := s.GetV1ApisAPIIDKeys(ctx, apiID, limit, offset, ownerID)
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
| `offset`                                              | **float64*                                            | :heavy_minus_sign:                                    | N/A                                                   |                                                       |
| `ownerID`                                             | **string*                                             | :heavy_minus_sign:                                    | N/A                                                   |                                                       |


### Response

**[*operations.GetV1ApisAPIIDKeysResponse](../../models/operations/getv1apisapiidkeysresponse.md), error**
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
