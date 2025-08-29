package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HelloResponse struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type DebugResponse struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Proto       string            `json:"proto"`
	Headers     map[string]string `json:"headers"`
	RawBody     string            `json:"raw_body"`
	ContentType string            `json:"content_type"`
	UserAgent   string            `json:"user_agent"`
	RemoteAddr  string            `json:"remote_addr"`
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type GreetingResponse struct {
	Greeting   string    `json:"greeting"`
	UserName   string    `json:"user_name"`
	Timestamp  time.Time `json:"timestamp"`
	APIVersion string    `json:"api_version"`
}

type Account struct {
	ID               string                 `json:"id"`
	EmailAddress     string                 `json:"email_address"`
	FullName         string                 `json:"full_name"`
	AccountType      string                 `json:"account_type"`
	Status           string                 `json:"status"`
	SubscriptionTier string                 `json:"subscription_tier"`
	CreatedTimestamp time.Time              `json:"created_timestamp"`
	LastModified     time.Time              `json:"last_modified"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type CreateAccountRequest struct {
	EmailAddress     string `json:"email_address"`
	FullName         string `json:"full_name"`
	AccountType      string `json:"account_type,omitempty"`
	SubscriptionTier string `json:"subscription_tier,omitempty"`
	PhoneNumber      string `json:"phone_number"`
	TermsAccepted    bool   `json:"terms_accepted"`
	MarketingConsent bool   `json:"marketing_consent,omitempty"`
}

type UpdateAccountRequest struct {
	EmailAddress     *string `json:"email_address,omitempty"`
	FullName         *string `json:"full_name,omitempty"`
	AccountType      *string `json:"account_type,omitempty"`
	Status           *string `json:"status,omitempty"`
	SubscriptionTier *string `json:"subscription_tier,omitempty"`
	PhoneNumber      *string `json:"phone_number,omitempty"`
	MarketingConsent *bool   `json:"marketing_consent,omitempty"`
}

type AccountListResponse struct {
	Accounts      []Account `json:"accounts"`
	NextPageToken string    `json:"next_page_token,omitempty"`
	TotalCount    int       `json:"total_count,omitempty"`
}

type PermissionsResponse struct {
	Permissions   []string `json:"permissions"`
	InheritedFrom *string  `json:"inherited_from,omitempty"`
}

type Error struct {
	Message   string    `json:"message"`
	ErrorCode string    `json:"error_code"`
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

type ValidationError struct {
	Message   string             `json:"message"`
	Details   []ValidationDetail `json:"details"`
	RequestID string             `json:"request_id"`
}

type ValidationDetail struct {
	Field string `json:"field"`
	Error string `json:"error"`
	Code  string `json:"code"`
}

var (
	accountStore = make(map[string]*Account)
	accountMutex sync.RWMutex
	accountIDPattern = regexp.MustCompile(`^acc_[a-zA-Z0-9]{20}$`)
)

func generateAccountID() string {
	bytes := make([]byte, 10)
	rand.Read(bytes)
	return "acc_" + hex.EncodeToString(bytes)
}

func generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "req_" + hex.EncodeToString(bytes)
}

func respondWithError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Error{
		Message:   message,
		ErrorCode: errorCode,
		RequestID: generateRequestID(),
		Timestamp: time.Now().UTC(),
	})
}

func respondWithValidationError(w http.ResponseWriter, message string, details []ValidationDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(ValidationError{
		Message:   message,
		Details:   details,
		RequestID: generateRequestID(),
	})
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

	// Debug endpoint - dumps request headers and body
	mux.HandleFunc("/v1/debug", func(w http.ResponseWriter, r *http.Request) {
		// Read body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// Convert headers to map
		headers := make(map[string]string)
		for key, values := range r.Header {
			// Join multiple values with comma
			headers[key] = strings.Join(values, ", ")
		}

		response := DebugResponse{
			Method:      r.Method,
			URL:         r.URL.String(),
			Proto:       r.Proto,
			Headers:     headers,
			RawBody:     string(bodyBytes),
			ContentType: r.Header.Get("Content-Type"),
			UserAgent:   r.Header.Get("User-Agent"),
			RemoteAddr:  r.RemoteAddr,
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cookie", "123=123")
		w.Header().Set("X-Custom-Header", "CustomValue")
		w.Header().Set("X-Custom-Header", "CustomValue")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
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

	mux.HandleFunc("/v1/timeout", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 35)
	})

	mux.HandleFunc("/v1/protected", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		if auth == "" || auth != "Bearer 123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	// V2 API Endpoints
	// Health endpoint
	mux.HandleFunc("/v2/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
			return
		}
		
		response := HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().UTC(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	
	// Greeting endpoint
	mux.HandleFunc("/v2/greeting", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
			return
		}
		
		name := r.URL.Query().Get("name")
		if name == "" {
			respondWithError(w, http.StatusBadRequest, "MISSING_PARAMETER", "Missing or invalid name parameter")
			return
		}
		
		response := GreetingResponse{
			Greeting:   fmt.Sprintf("Hello, %s!", name),
			UserName:   name,
			Timestamp:  time.Now().UTC(),
			APIVersion: "2.0.0",
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	
	// Accounts endpoints
	mux.HandleFunc("/v2/accounts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Get all accounts
			pageSize := 20
			if ps := r.URL.Query().Get("page_size"); ps != "" {
				if size, err := strconv.Atoi(ps); err == nil && size >= 1 && size <= 50 {
					pageSize = size
				}
			}
			
			pageToken := r.URL.Query().Get("page_token")
			status := r.URL.Query().Get("status")
			
			accountMutex.RLock()
			accounts := make([]Account, 0)
			for _, acc := range accountStore {
				if status != "" && acc.Status != status {
					continue
				}
				accounts = append(accounts, *acc)
			}
			accountMutex.RUnlock()
			
			// Simple pagination - in production would use proper cursor
			start := 0
			if pageToken != "" {
				if idx, err := strconv.Atoi(pageToken); err == nil {
					start = idx
				}
			}
			
			end := start + pageSize
			if end > len(accounts) {
				end = len(accounts)
			}
			
			var nextPageToken string
			if end < len(accounts) {
				nextPageToken = strconv.Itoa(end)
			}
			
			response := AccountListResponse{
				Accounts:      accounts[start:end],
				NextPageToken: nextPageToken,
				TotalCount:    len(accounts),
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			
		case http.MethodPost:
			// Create account
			var req CreateAccountRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondWithError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
				return
			}
			
			// Validate required fields
			var validationErrors []ValidationDetail
			
			if req.EmailAddress == "" || !strings.Contains(req.EmailAddress, "@") {
				validationErrors = append(validationErrors, ValidationDetail{
					Field: "email_address",
					Error: "Must be a valid email address",
					Code:  "INVALID_EMAIL",
				})
			}
			
			if req.FullName == "" || len(req.FullName) < 2 || len(req.FullName) > 100 {
				validationErrors = append(validationErrors, ValidationDetail{
					Field: "full_name",
					Error: "Must be between 2 and 100 characters",
					Code:  "INVALID_NAME",
				})
			}
			
			if req.PhoneNumber == "" || !strings.HasPrefix(req.PhoneNumber, "+") {
				validationErrors = append(validationErrors, ValidationDetail{
					Field: "phone_number",
					Error: "Must be a valid phone number with country code",
					Code:  "INVALID_PHONE",
				})
			}
			
			if !req.TermsAccepted {
				validationErrors = append(validationErrors, ValidationDetail{
					Field: "terms_accepted",
					Error: "Terms must be accepted",
					Code:  "TERMS_NOT_ACCEPTED",
				})
			}
			
			if len(validationErrors) > 0 {
				respondWithValidationError(w, "Validation failed", validationErrors)
				return
			}
			
			// Check for duplicate email
			accountMutex.RLock()
			for _, acc := range accountStore {
				if acc.EmailAddress == req.EmailAddress {
					accountMutex.RUnlock()
					respondWithError(w, http.StatusConflict, "ACCOUNT_EXISTS", "Account already exists")
					return
				}
			}
			accountMutex.RUnlock()
			
			// Set defaults
			if req.AccountType == "" {
				req.AccountType = "standard"
			}
			if req.SubscriptionTier == "" {
				req.SubscriptionTier = "free"
			}
			
			// Create account
			account := &Account{
				ID:               generateAccountID(),
				EmailAddress:     req.EmailAddress,
				FullName:         req.FullName,
				AccountType:      req.AccountType,
				Status:           "active",
				SubscriptionTier: req.SubscriptionTier,
				CreatedTimestamp: time.Now().UTC(),
				LastModified:     time.Now().UTC(),
				Metadata: map[string]interface{}{
					"source":           "api",
					"marketing_consent": req.MarketingConsent,
					"phone_number":      req.PhoneNumber,
				},
			}
			
			accountMutex.Lock()
			accountStore[account.ID] = account
			accountMutex.Unlock()
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(account)
			
		default:
			respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		}
	})
	
	// Account by ID endpoints
	mux.HandleFunc("/v2/accounts/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v2/accounts/")
		parts := strings.Split(path, "/")
		
		if len(parts) == 0 || parts[0] == "" {
			respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Not found")
			return
		}
		
		accountID := parts[0]
		
		// Validate account ID format
		if !accountIDPattern.MatchString(accountID) {
			respondWithError(w, http.StatusBadRequest, "INVALID_ACCOUNT_ID", "Invalid account ID format")
			return
		}
		
		// Handle permissions endpoint
		if len(parts) > 1 && parts[1] == "permissions" {
			if r.Method != http.MethodGet {
				respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
				return
			}
			
			accountMutex.RLock()
			account, exists := accountStore[accountID]
			accountMutex.RUnlock()
			
			if !exists {
				respondWithError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "Account not found")
				return
			}
			
			// Mock permissions based on account type
			var permissions []string
			var inheritedFrom *string
			
			switch account.AccountType {
			case "premium":
				permissions = []string{"read", "write", "delete", "admin", "billing"}
				roleType := "premium_role"
				inheritedFrom = &roleType
			case "standard":
				permissions = []string{"read", "write"}
				roleType := "standard_role"
				inheritedFrom = &roleType
			case "basic":
				permissions = []string{"read"}
				roleType := "basic_role"
				inheritedFrom = &roleType
			}
			
			response := PermissionsResponse{
				Permissions:   permissions,
				InheritedFrom: inheritedFrom,
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
		
		// Handle account CRUD operations
		switch r.Method {
		case http.MethodGet:
			accountMutex.RLock()
			account, exists := accountStore[accountID]
			accountMutex.RUnlock()
			
			if !exists {
				respondWithError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "Account not found")
				return
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(account)
			
		case http.MethodPatch:
			var req UpdateAccountRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondWithError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
				return
			}
			
			accountMutex.Lock()
			account, exists := accountStore[accountID]
			if !exists {
				accountMutex.Unlock()
				respondWithError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "Account not found")
				return
			}
			
			// Validate and apply updates
			var validationErrors []ValidationDetail
			
			if req.EmailAddress != nil {
				if !strings.Contains(*req.EmailAddress, "@") {
					validationErrors = append(validationErrors, ValidationDetail{
						Field: "email_address",
						Error: "Must be a valid email address",
						Code:  "INVALID_EMAIL",
					})
				} else {
					account.EmailAddress = *req.EmailAddress
				}
			}
			
			if req.FullName != nil {
				if len(*req.FullName) < 2 || len(*req.FullName) > 100 {
					validationErrors = append(validationErrors, ValidationDetail{
						Field: "full_name",
						Error: "Must be between 2 and 100 characters",
						Code:  "INVALID_NAME",
					})
				} else {
					account.FullName = *req.FullName
				}
			}
			
			if req.Status != nil && (*req.Status == "pending") {
				validationErrors = append(validationErrors, ValidationDetail{
					Field: "status",
					Error: "Cannot set status to pending",
					Code:  "INVALID_STATUS",
				})
			}
			
			if len(validationErrors) > 0 {
				accountMutex.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				json.NewEncoder(w).Encode(ValidationError{
					Message:   "Invalid update data",
					Details:   validationErrors,
					RequestID: generateRequestID(),
				})
				return
			}
			
			// Apply other updates
			if req.AccountType != nil {
				account.AccountType = *req.AccountType
			}
			if req.Status != nil {
				account.Status = *req.Status
			}
			if req.SubscriptionTier != nil {
				account.SubscriptionTier = *req.SubscriptionTier
			}
			if req.PhoneNumber != nil {
				if account.Metadata == nil {
					account.Metadata = make(map[string]interface{})
				}
				account.Metadata["phone_number"] = *req.PhoneNumber
			}
			if req.MarketingConsent != nil {
				if account.Metadata == nil {
					account.Metadata = make(map[string]interface{})
				}
				account.Metadata["marketing_consent"] = *req.MarketingConsent
			}
			
			account.LastModified = time.Now().UTC()
			accountMutex.Unlock()
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(account)
			
		case http.MethodDelete:
			accountMutex.Lock()
			_, exists := accountStore[accountID]
			if !exists {
				accountMutex.Unlock()
				respondWithError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "Account not found")
				return
			}
			
			delete(accountStore, accountID)
			accountMutex.Unlock()
			
			w.WriteHeader(http.StatusNoContent)
			
		default:
			respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		}
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
    email: support@unkey.dev
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
