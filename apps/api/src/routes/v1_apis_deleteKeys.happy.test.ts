import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { expect, test } from "vitest";
import type { V1ApisDeleteKeysRequest, V1ApisDeleteKeysResponse } from "./v1_apis_deleteKeys";

test.only("deletes the keys", async (t) => {
  const h = await IntegrationHarness.init(t);

  const n = 10;
  for (let i = 0; i < n; i++) {
    await h.createKey();
  }

  const apiId = h.resources.userApi.id;
  const root = await h.createRootKey([`api.${apiId}.delete_key`]);
  const softDeleteRes = await h.post<V1ApisDeleteKeysRequest, V1ApisDeleteKeysResponse>({
    url: "/v1/apis.deleteKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId,
      permanent: false,
    },
  });

  expect(softDeleteRes.status, `expected 200, received: ${JSON.stringify(softDeleteRes)}`).toBe(
    200,
  );
  expect(softDeleteRes.body.deletedKeys).toEqual(n);

  const apiBeforeHardDelete = await h.db.readonly.query.apis.findFirst({
    where: (table, { eq }) => eq(table.id, h.resources.userApi.id),
    with: {
      keyAuth: {
        with: {
          keys: true,
        },
      },
    },
  });
  expect(apiBeforeHardDelete).toBeDefined();
  expect(apiBeforeHardDelete!.keyAuth!.keys.length).toEqual(n);
  for (const k of apiBeforeHardDelete!.keyAuth!.keys) {
    expect(k.deletedAt).not.toBeNull();
  }

  await h.createKey();

  const hardDeleteRes = await h.post<V1ApisDeleteKeysRequest, V1ApisDeleteKeysResponse>({
    url: "/v1/apis.deleteKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId,
      permanent: true,
    },
  });

  expect(hardDeleteRes.status, `expected 200, received: ${JSON.stringify(hardDeleteRes)}`).toBe(
    200,
  );
  expect(hardDeleteRes.body.deletedKeys).toEqual(n + 1);

  const apiAfterHardDelete = await h.db.readonly.query.apis.findFirst({
    where: (table, { eq }) => eq(table.id, h.resources.userApi.id),
    with: {
      keyAuth: {
        with: {
          keys: true,
        },
      },
    },
  });
  expect(apiAfterHardDelete).toBeDefined();
  expect(apiAfterHardDelete!.keyAuth!.keys.length).toEqual(0);
});
