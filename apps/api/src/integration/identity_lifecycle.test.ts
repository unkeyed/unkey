import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { expect, test } from "vitest";

test("create new identity, update it, add ratelimits and verify associated keys", async (t) => {
  /**
   * Test setup
   */
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([
    "identity.*.create_identity",
    "identity.*.read_identity",
    "identity.*.update_identity",
    "api.*.create_key",
  ]);

  /**
   * You can get the `apiId` from the dashboard. Here we use a test id
   */
  const apiId = h.resources.userApi.id;

  /**
   * Use a userId or orgId from your system
   */
  const externalId = "user_1234abc";

  const createIdentityResponse = await fetch(`${h.baseUrl}/v1/identities.createIdentity`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: JSON.stringify({
      externalId,
      meta: {
        stripeCustomerId: "cus_123",
      },
    }),
  });

  const { identityId } = await createIdentityResponse.json<{
    identityId: string;
  }>();

  /**
   * Let's retrieve the identity to make sure it got created successfully
   */
  const getIdentityResponse = await fetch(
    `${h.baseUrl}/v1/identities.getIdentity?identityId=${identityId}`,
    {
      method: "GET",
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    },
  );

  const identity = await getIdentityResponse.json<{
    id: string;
    externalId: string;
    meta: unknown;
    ratelimits: Array<{ name: string; limit: number; duration: number }>;
  }>();

  expect(identity.externalId).toBe(externalId);
  expect(identity.meta).toMatchObject({
    stripeCustomerId: "cus_123",
  });

  /**
   * Let's create a key and connect it to the identity
   */

  const createKeyResponse = await fetch(`${h.baseUrl}/v1/keys.createKey`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: JSON.stringify({
      apiId: apiId,
      prefix: "acme",
      externalId: externalId,
    }),
  });

  const key = await createKeyResponse.json<{
    keyId: string;
    key: string;
  }>();

  /**
   * Let's verify the key and see how the identity is returned in the response
   */
  const verifyKeyResponse = await fetch(`${h.baseUrl}/v1/keys.verifyKey`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      apiId: apiId,
      key: key.key,
    }),
  });

  const verified = await verifyKeyResponse.json<{
    valid: boolean;
    identity: {
      id: string;
      externalId: string;
      meta: unknown;
    };
  }>();

  expect(verified.valid).toBe(true);
  expect(verified.identity.externalId).toBe(externalId);

  /**
   * Let's add some ratelimits
   */
  const updateRes = await fetch(`${h.baseUrl}/v1/identities.updateIdentity`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: JSON.stringify({
      identityId: identity.id,
      ratelimits: [
        /**
         * We define a limit that allows 10 requests per day
         */
        {
          name: "requests",
          limit: 10,
          duration: 24 * 60 * 60 * 1000, // 24h
        },
        /**
         * And a second limit that allows 1000 tokens per minute
         */
        {
          name: "tokens",
          limit: 1000,
          duration: 60 * 1000, // 1 minute
        },
      ],
    }),
  });
  expect(updateRes.status, `expected 200, got: ${JSON.stringify(updateRes)}`).toBe(200);

  /**
   * Now let's verify the key again and specify the limits
   *
   * In this case, we pretend like a user is requesting to use 200 tokens
   */
  const verifiedWithRatelimitsResponse = await fetch(`${h.baseUrl}/v1/keys.verifyKey`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      apiId: apiId,
      key: key.key,
      ratelimits: [
        {
          name: "requests",
        },
        {
          name: "tokens",
          cost: 200,
        },
      ],
    }),
  });

  const body = await verifiedWithRatelimitsResponse.text();
  expect(
    verifiedWithRatelimitsResponse.status,
    `expected 200, got: ${verifiedWithRatelimitsResponse.status} - ${body}`,
  ).toBe(200);

  const verifiedWithRatelimits = JSON.parse(body) as {
    valid: boolean;
    identity: {
      id: string;
      externalId: string;
      meta: unknown;
    };
  };

  expect(verifiedWithRatelimits.valid).toBe(true);
  expect(verifiedWithRatelimits.identity.externalId).toBe(externalId);
});
