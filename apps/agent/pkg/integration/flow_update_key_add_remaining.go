package integration

import (
	"context"
	"fmt"
)

var UpdateRemaining = newScenario(
	"UpdateRemaining",
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
			Assertions: []Asserter{
				AssertStatus(200),
				AssertBodyExists("apiId"),
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
			Assertions: []Asserter{
				AssertStatus(200),
			},
		}.Run(ctx, make(map[string]any))

		keyResponse := Step[map[string]any]{
			Name:   "Create Key",
			Method: "POST",
			Url:    fmt.Sprintf("%s/v1/keys.createKey", env.BaseUrl),
			Header: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", env.RootKey),
			},
			Body: map[string]any{
				"apiId":     createApiResponse["apiId"],
				"remaining": 5,
			},
			Assertions: []Asserter{
				AssertStatus(200),
				AssertBodyExists("key"),
				AssertBodyExists("keyId"),
			},
		}.Run(ctx, make(map[string]any))

		// Use up all 5 verifications
		for i := 4; i >= 0; i-- {

			Step[map[string]any]{
				Name:   "Verify Key",
				Method: "POST",
				Url:    fmt.Sprintf("%s/v1/keys.verifyKey", env.BaseUrl),
				Header: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"key": keyResponse["key"],
				},
				Assertions: []Asserter{
					AssertStatus(200),
					AssertBody("valid", true),
					AssertBody("remaining", float64(i)),
				},
			}.Run(ctx, make(map[string]any))
		}
		Step[map[string]any]{
			Name:   "Verify Key - should fail",
			Method: "POST",
			Url:    fmt.Sprintf("%s/v1/keys.verifyKey", env.BaseUrl),
			Header: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]any{
				"key": keyResponse["key"],
			},
			Assertions: []Asserter{
				AssertStatus(200),
				AssertBody("valid", false),
				AssertBody("code", "KEY_USAGE_EXCEEDED"),
				AssertBody("remaining", float64(0)),
			},
		}.Run(ctx, make(map[string]any))

		// Add 5 more verifications
		Step[map[string]any]{
			Name:   "Set remaining to 5",
			Method: "POST",
			Url:    fmt.Sprintf("%s/v1/keys.updateKey", env.BaseUrl),
			Header: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", env.RootKey),
			},
			Body: map[string]any{
				"keyId":     keyResponse["keyId"],
				"remaining": 5,
			},
			Assertions: []Asserter{
				AssertStatus(200),
			},
		}.Run(ctx, make(map[string]any))

		// Verify the key has new remaining
		Step[map[string]any]{
			Name:   "Verify key after update",
			Method: "POST",
			Url:    fmt.Sprintf("%s/v1/keys.verifyKey", env.BaseUrl),
			Header: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]any{
				"key": keyResponse["key"],
			},
			Assertions: []Asserter{
				AssertStatus(200),
				AssertBody("valid", true),
				AssertBody("remaining", float64(4)),
			},
		}.Run(ctx, make(map[string]any))

		// Delete the key
		Step[map[string]any]{
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
			Assertions: []Asserter{
				AssertStatus(200),
			},
		}.Run(ctx, make(map[string]any))

	})
