package validation

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// testOpenAPISpec is a minimal OpenAPI 3.1 spec for testing edge cases
const testOpenAPISpec = `
openapi: "3.1.0"
info:
  title: Test API
  version: "1.0.0"

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
  schemas:
    User:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: string
        name:
          type: string
        email:
          type: string
          format: email
      additionalProperties: false

    NullableString:
      type:
        - string
        - "null"

    OneOfExample:
      oneOf:
        - type: object
          required: [type, stringValue]
          properties:
            type:
              const: "string"
            stringValue:
              type: string
          additionalProperties: false
        - type: object
          required: [type, numberValue]
          properties:
            type:
              const: "number"
            numberValue:
              type: number
          additionalProperties: false

    AnyOfExample:
      anyOf:
        - type: object
          properties:
            name:
              type: string
        - type: object
          properties:
            title:
              type: string

    AllOfExample:
      allOf:
        - type: object
          properties:
            id:
              type: string
        - type: object
          properties:
            name:
              type: string
      required:
        - id
        - name

    ArrayOfIntegers:
      type: array
      items:
        type: integer
      minItems: 1
      maxItems: 10

    NestedObject:
      type: object
      required:
        - level1
      properties:
        level1:
          type: object
          required:
            - level2
          properties:
            level2:
              type: object
              required:
                - value
              properties:
                value:
                  type: string

    EnumExample:
      type: string
      enum:
        - active
        - inactive
        - pending

    PatternExample:
      type: string
      pattern: "^[a-z]+_[0-9]+$"

    TupleExample:
      type: array
      prefixItems:
        - type: string
          description: First element must be string
        - type: integer
          description: Second element must be integer
        - type: boolean
          description: Third element must be boolean
      minItems: 3
      maxItems: 3
      items: false

    TupleWithAdditionalItems:
      type: array
      prefixItems:
        - type: string
        - type: integer
      items:
        type: boolean

    Dog:
      type: object
      required:
        - petType
        - bark
      properties:
        petType:
          type: string
          const: "dog"
        bark:
          type: boolean
      additionalProperties: false

    Cat:
      type: object
      required:
        - petType
        - meow
      properties:
        petType:
          type: string
          const: "cat"
        meow:
          type: boolean
      additionalProperties: false

    Pet:
      oneOf:
        - $ref: "#/components/schemas/Dog"
        - $ref: "#/components/schemas/Cat"
      discriminator:
        propertyName: petType
        mapping:
          dog: "#/components/schemas/Dog"
          cat: "#/components/schemas/Cat"

    MinMaxExample:
      type: object
      properties:
        count:
          type: integer
          minimum: 0
          maximum: 100
        name:
          type: string
          minLength: 1
          maxLength: 50

security:
  - bearerAuth: []

paths:
  /test/user:
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/User"
      responses:
        "200":
          description: OK

  /test/nullable:
    post:
      operationId: testNullable
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                value:
                  $ref: "#/components/schemas/NullableString"
              required:
                - value
      responses:
        "200":
          description: OK

  /test/oneof:
    post:
      operationId: testOneOf
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/OneOfExample"
      responses:
        "200":
          description: OK

  /test/anyof:
    post:
      operationId: testAnyOf
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/AnyOfExample"
      responses:
        "200":
          description: OK

  /test/allof:
    post:
      operationId: testAllOf
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/AllOfExample"
      responses:
        "200":
          description: OK

  /test/array:
    post:
      operationId: testArray
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ArrayOfIntegers"
      responses:
        "200":
          description: OK

  /test/nested:
    post:
      operationId: testNested
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/NestedObject"
      responses:
        "200":
          description: OK

  /test/enum:
    post:
      operationId: testEnum
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - status
              properties:
                status:
                  $ref: "#/components/schemas/EnumExample"
      responses:
        "200":
          description: OK

  /test/pattern:
    post:
      operationId: testPattern
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - code
              properties:
                code:
                  $ref: "#/components/schemas/PatternExample"
      responses:
        "200":
          description: OK

  /test/minmax:
    post:
      operationId: testMinMax
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/MinMaxExample"
      responses:
        "200":
          description: OK

  /test/params/{id}:
    get:
      operationId: testParams
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
        - name: limit
          in: query
          required: false
          schema:
            type: integer
            minimum: 1
            maximum: 100
        - name: tags
          in: query
          required: false
          style: form
          explode: true
          schema:
            type: array
            items:
              type: string
        - name: X-Request-ID
          in: header
          required: false
          schema:
            type: string
            format: uuid
      responses:
        "200":
          description: OK

  /test/no-auth:
    post:
      operationId: testNoAuth
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                data:
                  type: string
      responses:
        "200":
          description: OK

  /test/inline-schema:
    post:
      operationId: testInlineSchema
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - name
                - age
              properties:
                name:
                  type: string
                  minLength: 1
                age:
                  type: integer
                  minimum: 0
                  maximum: 150
                tags:
                  type: array
                  items:
                    type: string
                  maxItems: 5
              additionalProperties: false
      responses:
        "200":
          description: OK

  /test/ref-with-siblings:
    post:
      operationId: testRefWithSiblings
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
                - $ref: "#/components/schemas/User"
                - type: object
                  properties:
                    createdAt:
                      type: string
                      format: date-time
      responses:
        "200":
          description: OK

  /test/tuple:
    post:
      operationId: testTuple
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/TupleExample"
      responses:
        "200":
          description: OK

  /test/tuple-additional:
    post:
      operationId: testTupleAdditional
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/TupleWithAdditionalItems"
      responses:
        "200":
          description: OK

  /test/pet:
    post:
      operationId: testPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Pet"
      responses:
        "200":
          description: OK

  /test/pet-inline:
    post:
      operationId: testPetInline
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - $ref: "#/components/schemas/Dog"
                - $ref: "#/components/schemas/Cat"
              discriminator:
                propertyName: petType
                mapping:
                  dog: "#/components/schemas/Dog"
                  cat: "#/components/schemas/Cat"
      responses:
        "200":
          description: OK
`

// newTestValidator creates a validator from the test spec
func newTestValidator(t *testing.T) *Validator {
	t.Helper()

	parser, err := NewSpecParser([]byte(testOpenAPISpec))
	require.NoError(t, err, "failed to parse test spec")

	compiler, err := NewSchemaCompiler(parser, []byte(testOpenAPISpec))
	require.NoError(t, err, "failed to compile schemas")

	matcher := NewPathMatcher(parser.Operations())

	return &Validator{
		matcher:         matcher,
		compiler:        compiler,
		securitySchemes: parser.SecuritySchemes(),
	}
}

func makeRequest(method, path, body string, headers map[string]string) *http.Request {
	var bodyReader *bytes.Reader
	if body != "" {
		bodyReader = bytes.NewReader([]byte(body))
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req
}
