package integration

import (
	"context"
	"fmt"
)

var ListKeys = newScenario(
	"ListKeys",
	func(ctx context.Context, env Env) {
		createApiResponse := Step[map[string]any]{
			Name:   "Create API",
			Method: "POST",
			Url:    fmt.Sprintf("%s/v1/apis.createApi", env.BaseUrl),
			Header: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", env.RootKey),
			},
			Body: map[string]any{
				"name": "scenario-test-pls-delete",
			},
			Assertions: []assertion{
				assertStatus(200),
				assertBodyExists("apiId"),
				assertHeaderExists("Unkey-Trace-Id"),
			},
		}.Run(ctx, make(map[string]any))

		defer Step[map[string]any]{
			Name:   "Delete API",
			Method: "POST",
			Url:    fmt.Sprintf("%s/v1/apis.removeApi", env.BaseUrl),
			Header: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", env.RootKey),
			},
			Body: map[string]any{
				"apiId": createApiResponse["apiId"],
			},
			Assertions: []assertion{
				assertStatus(200),
				assertHeaderExists("Unkey-Trace-Id"),
			},
		}.Run(ctx, make(map[string]any))

		// Create 5 keys
		for i := 0; i < 5; i++ {
			keyResponse := Step[map[string]any]{
				Name:   "Create Key",
				Method: "POST",
				Url:    fmt.Sprintf("%s/v1/keys.createKey", env.BaseUrl),
				Header: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": fmt.Sprintf("Bearer %s", env.RootKey),
				},
				Body: map[string]any{
					"apiId": createApiResponse["apiId"],
				},
				Assertions: []assertion{
					assertStatus(200),
					assertBodyExists("key"),
					assertBodyExists("keyId"),
					assertHeaderExists("Unkey-Trace-Id"),
				},
			}.Run(ctx, make(map[string]any))

			defer Step[map[string]any]{
				Name:   "Revoke Key",
				Method: "POST",
				Url:    fmt.Sprintf("%s/v1/keys.removeKey", env.BaseUrl),
				Header: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": fmt.Sprintf("Bearer %s", env.RootKey),
				},
				Body: map[string]any{
					"keyId": keyResponse["keyId"],
				},
				Assertions: []assertion{
					assertStatus(200),
					assertHeaderExists("Unkey-Trace-Id"),
				},
			}.Run(ctx, make(map[string]any))

		}

		listKeys := Step[map[string]any]{
			Name:   "List Keys",
			Method: "GET",
			Url:    fmt.Sprintf("%s/v1/apis/%s/keys", env.BaseUrl, createApiResponse["apiId"]),
			Header: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", env.RootKey),
			},
			Assertions: []assertion{
				assertStatus(200),
				assertBodyExists("keys"),
				assertHeaderExists("Unkey-Trace-Id"),
			},
		}
		foundKeys := listKeys.Run(ctx, make(map[string]any))

		if len(foundKeys["keys"].([]any)) != 5 {
			listKeys.fail("expected 5 keys, got %d", len(foundKeys["keys"].([]any)))
		}

	})
