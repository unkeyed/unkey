# Unkey OpenAPI Specification

This directory contains the Unkey API OpenAPI specification split into multiple files for better maintainability.

## Structure

```
openapi/
├── openapi-split.yaml      # Main entry point with info, servers, security
└── spec/                   # All specification files
    ├── paths/              # Path definitions organized by API version
    │   └── v2/            # Version 2 API endpoints
    │       └── keys/      # Key management endpoints
    │           ├── setPermissions/
    │           │   ├── index.yaml    # Main path definition
    │           │   ├── request.yaml  # Request body & examples
    │           │   └── response.yaml # Response & examples
    │           └── updateKey/
    │               ├── index.yaml
    │               ├── request.yaml
    │               └── response.yaml
    ├── common/             # Shared schemas
    │   ├── meta.yaml      # Meta response object
    │   └── pagination.yaml # Pagination object
    └── error/             # Error-related schemas
        ├── base.yaml      # Base error schema
        ├── errors.yaml    # Error detail schemas
        └── responses.yaml # Common error responses
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
2. Create three files in the new directory:
   - `index.yaml` - Main path definition with operation details
   - `request.yaml` - Request body definition with schema and examples
   - `response.yaml` - Response definition with schema and examples
3. Update `openapi-split.yaml` to include your new endpoint in the `paths` section
4. If needed, add shared schemas to `spec/common/` or `spec/error/`

## Example Endpoint Structure

See `spec/paths/v2/keys/setPermissions/` for a complete example:

```
spec/paths/v2/keys/setPermissions/
├── index.yaml     # Main operation definition
├── request.yaml   # Request body with schema and examples
└── response.yaml  # Response with schema and examples
```

### index.yaml
Contains the main operation definition with references to request and response files:
```yaml
post:
  summary: Set (replace) all permissions on an API key
  operationId: setPermissions
  requestBody:
    $ref: "./request.yaml#/Request"
  responses:
    "200":
      $ref: "./response.yaml#/Response"
    "400":
      $ref: "../../../../error/responses.yaml#/BadRequestError"
```

### request.yaml
Contains both the schema definition and request body with examples:
```yaml
V2KeysSetPermissionsRequestBody:
  type: object
  required:
    - keyId
    - permissions
  properties:
    keyId:
      type: string
      description: The unique identifier of the key
      example: key_2cGKbMxRyIzhCxo1Idjz8q

Request:
  required: true
  content:
    application/json:
      schema:
        $ref: "#/V2KeysSetPermissionsRequestBody"
      examples:
        basic:
          summary: Basic example
          value:
            keyId: key_123
            permissions: [...]
```

### response.yaml  
Contains both the schema definition and response with examples:
```yaml
V2KeysSetPermissionsResponse:
  type: object
  required:
    - meta
    - data
  properties:
    meta:
      $ref: "../../../../common/meta.yaml"
    data:
      type: array

Response:
  description: Success response
  content:
    application/json:
      schema:
        $ref: "#/V2KeysSetPermissionsResponse"
      examples:
        standard:
          summary: Standard response
          value:
            meta:
              requestId: req_123
            data: [...]
```

## Benefits of Split Structure

- **Better organization**: Related endpoints and schemas are grouped together
- **Easier collaboration**: Multiple developers can work on different endpoints without conflicts
- **Improved maintainability**: Changes to one endpoint don't affect others
- **Reusability**: Common schemas and responses are defined once and referenced
- **Version control**: Smaller files mean clearer diffs and easier code reviews