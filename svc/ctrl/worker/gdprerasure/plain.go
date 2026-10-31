package gdprerasure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// plainURL is Plain's GraphQL endpoint, the one the dashboard's SDK calls.
const plainURL = "https://core-api.uk.plain.com/graphql/v1"

// PlainCustomers deletes the Plain customers the dashboard creates when a user
// contacts support.
type PlainCustomers interface {
	// DeleteCustomers deletes every Plain customer with this email address or
	// external id and reports how many existed. Either may be empty.
	DeleteCustomers(ctx context.Context, email, externalID string) (int64, error)
}

type plainCustomers struct {
	apiKey string
	url    string
	client *http.Client
}

var _ PlainCustomers = (*plainCustomers)(nil)

// NewPlain builds a Plain-backed PlainCustomers from an API key. The key needs
// the customer:read and customer:delete permissions.
func NewPlain(apiKey string) PlainCustomers {
	return &plainCustomers{
		apiKey: apiKey,
		url:    plainURL,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

const (
	plainCustomerByEmail      = `query customerByEmail($email: String!) { customerByEmail(email: $email) { id } }`
	plainCustomerByExternalID = `query customerByExternalId($externalId: ID!) { customerByExternalId(externalId: $externalId) { id } }`
	plainDeleteCustomer       = `mutation deleteCustomer($input: DeleteCustomerInput!) { deleteCustomer(input: $input) { error { message type code } } }`
)

type plainRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type plainResponse struct {
	Data   plainData    `json:"data"`
	Errors []plainError `json:"errors"`
}

// plainData holds every field the client selects; a response fills the one
// its operation asked for.
type plainData struct {
	CustomerByEmail      *plainCustomer     `json:"customerByEmail"`
	CustomerByExternalID *plainCustomer     `json:"customerByExternalId"`
	DeleteCustomer       *plainDeleteOutput `json:"deleteCustomer"`
}

type plainCustomer struct {
	ID string `json:"id"`
}

type plainDeleteOutput struct {
	Error *plainError `json:"error"`
}

type plainError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// DeleteCustomers looks the customer up by both identifiers, because the
// dashboard creates customers by email with the WorkOS user id as external id,
// and a customer outlives a change of either. Plain deletes a customer's
// threads and timeline with it.
func (p *plainCustomers) DeleteCustomers(ctx context.Context, email, externalID string) (int64, error) {
	var ids []string
	if email != "" {
		data, err := p.do(ctx, plainCustomerByEmail, map[string]any{"email": email})
		if err != nil {
			return 0, fmt.Errorf("plain customerByEmail: %w", err)
		}
		if data.CustomerByEmail != nil {
			ids = append(ids, data.CustomerByEmail.ID)
		}
	}
	if externalID != "" {
		data, err := p.do(ctx, plainCustomerByExternalID, map[string]any{"externalId": externalID})
		if err != nil {
			return 0, fmt.Errorf("plain customerByExternalId: %w", err)
		}
		if data.CustomerByExternalID != nil && (len(ids) == 0 || ids[0] != data.CustomerByExternalID.ID) {
			ids = append(ids, data.CustomerByExternalID.ID)
		}
	}

	var deleted int64
	for _, id := range ids {
		data, err := p.do(ctx, plainDeleteCustomer, map[string]any{"input": map[string]string{"customerId": id}})
		if err != nil {
			return deleted, fmt.Errorf("plain deleteCustomer %s: %w", id, err)
		}
		if data.DeleteCustomer == nil {
			return deleted, fmt.Errorf("plain deleteCustomer %s: no result", id)
		}
		if mErr := data.DeleteCustomer.Error; mErr != nil {
			return deleted, fmt.Errorf("plain deleteCustomer %s: %s: %s", id, mErr.Code, mErr.Message)
		}
		deleted++
	}
	return deleted, nil
}

// do sends one GraphQL operation. Plain answers most failures with status 200
// and an errors list, so both count as failures.
func (p *plainCustomers) do(ctx context.Context, query string, variables map[string]any) (plainData, error) {
	var data plainData
	body, err := json.Marshal(plainRequest{Query: query, Variables: variables})
	if err != nil {
		return data, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return data, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := p.client.Do(req)
	if err != nil {
		return data, err
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return data, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return data, fmt.Errorf("status %d: %s", res.StatusCode, raw[:min(len(raw), 4096)])
	}
	var decoded plainResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return data, fmt.Errorf("decode response: %w", err)
	}
	if len(decoded.Errors) > 0 {
		messages := make([]error, 0, len(decoded.Errors))
		for _, gqlErr := range decoded.Errors {
			messages = append(messages, errors.New(gqlErr.Message))
		}
		return data, errors.Join(messages...)
	}
	return decoded.Data, nil
}
