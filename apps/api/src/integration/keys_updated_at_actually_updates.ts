import { IntegrationHarness } from "@/pkg/testutil/integration-harness";

import type { V1KeysGetKeyResponse } from "@/routes/v1_keys_getKey";
import type { V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse } from "@/routes/v1_keys_updateKey";
import { expect, test } from "vitest";

test(
  "updatedAt updates",
  async (t) => {
    const h = await IntegrationHarness.init(t);
    const { key: rootKey } = await h.createRootKey(["*"]);

    const { keyId } = await h.createKey();

    const keyInDb = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, keyId),
    });
    expect(keyInDb).toBeDefined();
    expect(keyInDb?.updatedAtM).toBeNull();

    const updateRes = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${rootKey}`,
      },
      body: {
        keyId: keyId,
        name: "updated",
      },
    });

    expect(updateRes.status).toBe(200);

    const keyInDbAfterUpdate = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, keyId),
    });
    expect(keyInDbAfterUpdate).toBeDefined();
    expect(keyInDbAfterUpdate?.updatedAtM).not.toBeNull();
    expect(keyInDbAfterUpdate?.updatedAtM).toBeGreaterThan(
      // biome-ignore lint/style/noNonNullAssertion: Safe to leave
      keyInDbAfterUpdate!.createdAtM,
    );

    const returnedKey = await h.get<V1KeysGetKeyResponse>({
      url: `/v1/keys.getKey?keyId=${keyId}`,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${rootKey}`,
      },
    });

    expect(returnedKey.status).toEqual(200);
    expect(returnedKey.body.updatedAt).toEqual(keyInDbAfterUpdate?.updatedAtM);
  },
  { timeout: 30_000 },
);
