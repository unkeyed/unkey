


import { describe, test, expect, afterAll } from "bun:test"
import { step } from "@/pkg/testutil/step"
import { testEnv } from "./env"
import { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi"



describe("List Keys", () => {
  const env = testEnv()


  const createApiResponse = await step<V1ApisCreateApiRequest, V1ApisCreateApiResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/apis.createApi`,
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
    },
    body: {
      name: "scenario-test-pls-delete",
    }
  })
  expect(createApiResponse.status).toEqual(200)
  expect(createApiResponse.body.apiId).toBeDefined()
  expect(createApiResponse.headers).toHaveProperty("unkey-trace-id")

  afterAll(async () => {
    const deleteApiResponse = await step({
      url: `${env.UNKEY_BASE_URL}/v1/apis.removeApi`,
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
      },
      body: {
        apiId: createApiResponse.body.apiId,
      }
    })
    expect(deleteApiResponse.status).toEqual(200)
    expect(deleteApiResponse.headers).toHaveProperty("unkey-trace-id")
  })


  for (let i = 0; i < 5; i++) {
    const keyResponse = await step < V1Cr({})
  }

})

// Create 5 keys
for i := 0; i < 5; i++ {
  keyResponse:= Step[map[string]any]{
    Name: "Create Key",
      Method: "POST",
        Url: fmt.Sprintf("%s/v1/keys.createKey", env.BaseUrl),
          Header: map[string]string{
      "Content-Type": "application/json",
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
    Name: "Revoke Key",
      Method: "POST",
        Url: fmt.Sprintf("%s/v1/keys.removeKey", env.BaseUrl),
          Header: map[string]string{
      "Content-Type": "application/json",
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

listKeys:= Step[map[string]any]{
  Name: "List Keys",
    Method: "GET",
      Url: fmt.Sprintf("%s/v1/apis/%s/keys", env.BaseUrl, createApiResponse["apiId"]),
        Header: map[string]string{
    "Content-Type": "application/json",
      "Authorization": fmt.Sprintf("Bearer %s", env.RootKey),
			},
  Assertions: []assertion{
    assertStatus(200),
      assertBodyExists("keys"),
      assertHeaderExists("Unkey-Trace-Id"),
			},
}
foundKeys:= listKeys.Run(ctx, make(map[string]any))

if len(foundKeys["keys"].([]any)) != 5 {
  listKeys.fail("expected 5 keys, got %d", len(foundKeys["keys"].([]any)))
}

	})
