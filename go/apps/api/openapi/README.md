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
post:
  summary: List API keys
  operationId: listKeys
  requestBody:
    content:
      application/json:
        schema:
          $ref: "./V2ApisListKeysRequestBody.yaml"
  responses:
    "200":
      content:
        application/json:
          schema:
            $ref: "./V2ApisListKeysResponseBody.yaml"
    "400":
      content:
        application/json:
          schema:
            $ref: "../../../../error/BadRequestErrorResponse.yaml"
```

### V2ApisListKeysRequestBody.yaml

Contains the request body schema definition:

```yaml
type: object
required:
  - apiId
properties:
  apiId:
    type: string
    minLength: 1
    description: The ID of the API whose keys you want to list
    example: api_1234
  limit:
    type: integer
    description: Maximum number of keys to return
    default: 100
    minimum: 1
    maximum: 100
  cursor:
    type: string
    description: Pagination cursor from a previous response
    example: cursor_eyJsYXN0S2V5SWQiOiJrZXlfMjNld3MiLCJsYXN0Q3JlYXRlZEF0IjoxNjcyNTI0MjM0MDAwfQ==
```

### V2ApisListKeysResponseBody.yaml

Contains the response body schema definition:

```yaml
type: object
required:
  - meta
  - data
properties:
  meta:
    $ref: "../../../../common/Meta.yaml"
  data:
    $ref: "./V2ApisListKeysResponseData.yaml"
  pagination:
    $ref: "../../../../common/Pagination.yaml"
```

## Benefits of Split Structure

- **Better organization**: Related endpoints and schemas are grouped together
- **Easier collaboration**: Multiple developers can work on different endpoints without conflicts
- **Improved maintainability**: Changes to one endpoint don't affect others
- **Reusability**: Common schemas and responses are defined once and referenced
- **Version control**: Smaller files mean clearer diffs and easier code reviews
