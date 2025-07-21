# Unkey OpenAPI Specification

This directory contains the Unkey API OpenAPI specification split into multiple files for better maintainability.

## Structure

```
openapi/
├── openapi-split.yaml      # Main entry point with info, servers, security
└── spec/                   # All specification files
    ├── paths/              # Path definitions organized by API version
    │   └── v2/            # Version 2 API endpoints
    │       ├── apis/      # API management endpoints
    │       ├── identities/ # Identity management endpoints
    │       ├── keys/      # Key management endpoints
    │       ├── liveness/   # Health check endpoints
    │       ├── permissions/ # Permission and role management
    │       └── ratelimit/  # Rate limiting endpoints
    ├── common/             # Shared schemas (Meta.yaml, Pagination.yaml, etc.)
    └── error/             # Error-related schemas
```

## Usage

### Viewing the Split Specification

The split specification starts at `openapi-split.yaml`. Most OpenAPI tools support file references (`$ref`) and will automatically resolve them.

### Bundling into a Single File

To bundle all files into a single OpenAPI specification and generate Go code:

```bash
# Bundle the specification and generate Go code
go generate
```

This creates `openapi-bundled.yaml` with all references resolved using [libopenapi](https://pb33f.io/libopenapi/) and generates Go code as configured.

## Adding New Endpoints

1. Create a new directory under the appropriate category in `spec/paths/v2/` (e.g., `spec/paths/v2/keys/newEndpoint/`)
2. Create the necessary files in the new directory using explicit naming:
   - `index.yaml` - Main path definition with operation details
   - `V2CategoryOperationRequestBody.yaml` - Request body schema
   - `V2CategoryOperationResponseBody.yaml` - Response body schema
   - Additional data schemas as needed (e.g., `V2CategoryOperationResponseData.yaml`)
3. Update `openapi-split.yaml` to include your new endpoint in the `paths` section
4. If needed, add shared schemas to `spec/common/` or `spec/error/`

## Example Endpoint Structure

See `spec/paths/v2/apis/listKeys/` for a complete example:

```
spec/paths/v2/apis/listKeys/
├── index.yaml                        # Main operation definition
├── V2ApisListKeysRequestBody.yaml    # Request body schema
├── V2ApisListKeysResponseBody.yaml   # Response body schema
└── V2ApisListKeysResponseData.yaml   # Response data schema
```

### index.yaml

Contains the main operation definition with references to request and response files:

```yaml
get:
  summary: List API keys
  operationId: listKeys
  requestBody:
    $ref: "./V2ApisListKeysRequestBody.yaml#/RequestBody"
  responses:
    "200":
      $ref: "./V2ApisListKeysResponseBody.yaml#/ResponseBody"
    "400":
      $ref: "../../../../error/BadRequestErrorResponse.yaml#/BadRequestErrorResponse"
```

### V2ApisListKeysRequestBody.yaml

Contains the request body schema definition:

```yaml
V2ApisListKeysRequestBody:
  type: object
  required:
    - apiId
  properties:
    apiId:
      type: string
      description: The unique identifier of the API
      example: api_2cGKbMxRyIzhCxo1Idjz8q

RequestBody:
  required: true
  content:
    application/json:
      schema:
        $ref: "#/V2ApisListKeysRequestBody"
      examples:
        basic:
          summary: Basic example
          value:
            apiId: api_123
```

### V2ApisListKeysResponseBody.yaml

Contains the response body schema definition:

```yaml
V2ApisListKeysResponseBody:
  type: object
  required:
    - meta
    - data
  properties:
    meta:
      $ref: "../../../../common/Meta.yaml#/Meta"
    data:
      $ref: "./V2ApisListKeysResponseData.yaml#/V2ApisListKeysResponseData"

ResponseBody:
  description: Success response
  content:
    application/json:
      schema:
        $ref: "#/V2ApisListKeysResponseBody"
      examples:
        standard:
          summary: Standard response
          value:
            meta:
              requestId: req_123
            data:
              keys: [...]
```

## Benefits of Split Structure

- **Better organization**: Related endpoints and schemas are grouped together
- **Easier collaboration**: Multiple developers can work on different endpoints without conflicts
- **Improved maintainability**: Changes to one endpoint don't affect others
- **Reusability**: Common schemas and responses are defined once and referenced
- **Version control**: Smaller files mean clearer diffs and easier code reviews
