package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type HelloResponse struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/v1/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	// Hello endpoint
	mux.HandleFunc("/v1/hello", func(w http.ResponseWriter, r *http.Request) {
		response := HelloResponse{
			Message:   "Hello from demo API",
			Timestamp: time.Now().UTC(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// OpenAPI spec endpoint - VERSION 2 (Breaking Changes)
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		spec := `openapi: 3.1.0
info:
  title: Demo API
  description: A simple demo API for testing deployments with breaking changes
  version: 2.0.0
  contact:
    name: Unkey Support
    email: support@unkey.com
servers:
  - url: /v2
    description: API v2 (BREAKING CHANGES)
paths:
  /health:
    get:
      operationId: getHealth
      summary: Health check endpoint (renamed from liveness)
      description: Returns OK if the service is healthy
      responses:
        '200':
          description: Service is healthy
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: "healthy"
                  timestamp:
                    type: string
                    format: date-time
                required:
                  - status
                  - timestamp
  /greeting:
    get:
      operationId: getGreeting
      summary: Greeting endpoint (renamed from hello)
      description: Returns a greeting message with timestamp
      parameters:
        - name: name
          in: query
          description: Name to greet (new required parameter)
          required: true
          schema:
            type: string
            minLength: 1
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/GreetingResponse'
        '400':
          description: Missing or invalid name parameter
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
  /accounts:
    get:
      operationId: getAccounts
      summary: Get all accounts (renamed from users)
      description: Returns a list of user accounts
      parameters:
        - name: page_size
          in: query
          description: Number of accounts per page (renamed from limit)
          required: false
          schema:
            type: integer
            minimum: 1
            maximum: 50
            default: 20
        - name: page_token
          in: query
          description: Pagination token (changed from offset)
          required: false
          schema:
            type: string
        - name: status
          in: query
          description: Filter by account status (new parameter)
          required: false
          schema:
            type: string
            enum: [active, suspended, pending]
      responses:
        '200':
          description: List of accounts
          content:
            application/json:
              schema:
                type: object
                properties:
                  accounts:
                    type: array
                    items:
                      $ref: '#/components/schemas/Account'
                  next_page_token:
                    type: string
                    description: Token for next page
                  total_count:
                    type: integer
                    description: Total number of accounts
                required:
                  - accounts
    post:
      operationId: createAccount
      summary: Create a new account
      description: Creates a new user account with enhanced validation
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateAccountRequest'
      responses:
        '201':
          description: Account created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Account'
        '400':
          description: Invalid request
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ValidationError'
        '409':
          description: Account already exists
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
  /accounts/{accountId}:
    get:
      operationId: getAccountById
      summary: Get account by ID
      description: Returns a specific account by their ID
      parameters:
        - name: accountId
          in: path
          required: true
          description: The account ID (changed pattern)
          schema:
            type: string
            pattern: '^acc_[a-zA-Z0-9]{20}$'
      responses:
        '200':
          description: Account found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Account'
        '404':
          description: Account not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
    patch:
      operationId: updateAccount
      summary: Update account (changed from PUT to PATCH)
      description: Partially updates an existing account
      parameters:
        - name: accountId
          in: path
          required: true
          description: The account ID
          schema:
            type: string
            pattern: '^acc_[a-zA-Z0-9]{20}$'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateAccountRequest'
      responses:
        '200':
          description: Account updated successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Account'
        '404':
          description: Account not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
        '422':
          description: Invalid update data
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ValidationError'
    delete:
      operationId: deleteAccount
      summary: Delete account (new endpoint)
      description: Permanently deletes an account
      parameters:
        - name: accountId
          in: path
          required: true
          description: The account ID
          schema:
            type: string
            pattern: '^acc_[a-zA-Z0-9]{20}$'
      responses:
        '204':
          description: Account deleted successfully
        '404':
          description: Account not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
  /accounts/{accountId}/permissions:
    get:
      operationId: getAccountPermissions
      summary: Get account permissions (new endpoint)
      description: Returns permissions for a specific account
      parameters:
        - name: accountId
          in: path
          required: true
          description: The account ID
          schema:
            type: string
            pattern: '^acc_[a-zA-Z0-9]{20}$'
      responses:
        '200':
          description: Account permissions
          content:
            application/json:
              schema:
                type: object
                properties:
                  permissions:
                    type: array
                    items:
                      type: string
                  inherited_from:
                    type: [string, null]
                required:
                  - permissions
components:
  schemas:
    GreetingResponse:
      type: object
      properties:
        greeting:
          type: string
          description: The personalized greeting message
          example: "Hello, John!"
        user_name:
          type: string
          description: The name that was greeted
          example: "John"
        timestamp:
          type: string
          format: date-time
          description: The current timestamp
          example: "2023-12-07T10:30:00Z"
        api_version:
          type: string
          description: API version used
          example: "2.0.0"
      required:
        - greeting
        - user_name
        - timestamp
        - api_version
    Account:
      type: object
      properties:
        id:
          type: string
          description: Unique account identifier (changed pattern)
          pattern: '^acc_[a-zA-Z0-9]{20}$'
          example: "acc_1234567890abcdef1234"
        email_address:
          type: string
          format: email
          description: Account email address (renamed field)
          example: "john.doe@example.com"
        full_name:
          type: string
          description: Account holder's full name (renamed field)
          example: "John Doe"
        account_type:
          type: string
          enum: [premium, standard, basic]
          description: Account type (changed from role)
          example: "standard"
        status:
          type: string
          enum: [active, suspended, pending]
          description: Account status (new field)
          example: "active"
        subscription_tier:
          type: string
          enum: [free, pro, enterprise]
          description: Subscription tier (new field)
          example: "pro"
        created_timestamp:
          type: string
          format: date-time
          description: When the account was created (renamed field)
          example: "2023-12-07T10:30:00Z"
        last_modified:
          type: string
          format: date-time
          description: When the account was last modified (renamed field)
          example: "2023-12-07T10:30:00Z"
        metadata:
          type: object
          description: Additional account metadata (new field)
          additionalProperties: true
          example: {"source": "web", "referrer": "google"}
      required:
        - id
        - email_address
        - full_name
        - account_type
        - status
        - subscription_tier
        - created_timestamp
        - last_modified
    CreateAccountRequest:
      type: object
      properties:
        email_address:
          type: string
          format: email
          description: Account email address
          example: "john.doe@example.com"
        full_name:
          type: string
          description: Account holder's full name
          minLength: 2
          maxLength: 100
          example: "John Doe"
        account_type:
          type: string
          enum: [premium, standard, basic]
          description: Account type
          default: "standard"
          example: "standard"
        subscription_tier:
          type: string
          enum: [free, pro, enterprise]
          description: Subscription tier
          default: "free"
          example: "free"
        phone_number:
          type: string
          description: Phone number (new required field)
          pattern: '^\+[1-9]\d{1,14}$'
          example: "+1234567890"
        terms_accepted:
          type: boolean
          description: Whether terms of service were accepted (new required field)
          example: true
        marketing_consent:
          type: boolean
          description: Marketing consent flag (new field)
          default: false
          example: false
      required:
        - email_address
        - full_name
        - phone_number
        - terms_accepted
    UpdateAccountRequest:
      type: object
      properties:
        email_address:
          type: string
          format: email
          description: Account email address
          example: "john.doe@example.com"
        full_name:
          type: string
          description: Account holder's full name
          minLength: 2
          maxLength: 100
          example: "John Doe"
        account_type:
          type: string
          enum: [premium, standard, basic]
          description: Account type
          example: "premium"
        status:
          type: string
          enum: [active, suspended]
          description: Account status (pending cannot be set via update)
          example: "active"
        subscription_tier:
          type: string
          enum: [free, pro, enterprise]
          description: Subscription tier
          example: "pro"
        phone_number:
          type: string
          description: Phone number
          pattern: '^\+[1-9]\d{1,14}$'
          example: "+1234567890"
        marketing_consent:
          type: boolean
          description: Marketing consent flag
          example: true
    ValidationError:
      type: object
      properties:
        message:
          type: string
          description: Main error message
          example: "Validation failed"
        details:
          type: array
          items:
            type: object
            properties:
              field:
                type: string
                description: Field that failed validation
                example: "email_address"
              error:
                type: string
                description: Specific validation error
                example: "Must be a valid email address"
              code:
                type: string
                description: Error code
                example: "INVALID_EMAIL"
            required:
              - field
              - error
              - code
        request_id:
          type: string
          description: Request ID for tracking
          example: "req_1234567890abcdef"
      required:
        - message
        - details
        - request_id
    Error:
      type: object
      properties:
        message:
          type: string
          description: Error message (renamed from error)
          example: "Account not found"
        error_code:
          type: string
          description: Error code (renamed from code)
          example: "ACCOUNT_NOT_FOUND"
        request_id:
          type: string
          description: Request ID for tracking (new field)
          example: "req_1234567890abcdef"
        timestamp:
          type: string
          format: date-time
          description: When the error occurred (new field)
          example: "2023-12-07T10:30:00Z"
      required:
        - message
        - error_code
        - request_id
        - timestamp`

		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, spec)
	})

	log.Printf("Demo API starting on port %s", port)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
