package integration

import (
	"context"
	"fmt"
)

var CreateVerifyDeleteKeys = newScenario(
	"CreateVerifyDeleteKeys",
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
				Assertions: []Asserter{
					AssertStatus(200),
					AssertBodyExists("key"),
					AssertBodyExists("keyId"),
				},
			}.Run(ctx, make(map[string]any))

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
				},
			}.Run(ctx, make(map[string]any))

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

		}

	})
