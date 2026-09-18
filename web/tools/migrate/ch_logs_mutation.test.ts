import assert from "node:assert/strict";
import test from "node:test";
import { buildExternalIdBackfillMutation } from "./ch_logs_mutation";

const row = {
  workspace_id: "ws_123",
  key_space_id: "ks_456",
  identity_id: "id_789",
};

test("passes the external id through as a query parameter", () => {
  const { query, query_params } = buildExternalIdBackfillMutation(
    "default.key_verifications_per_month_v2",
    row,
    "user's-identity",
  );

  assert.equal(query_params.externalId, "user's-identity");
  assert.ok(query.includes("SET external_id = {externalId: String}"));
  assert.ok(!query.includes("user's-identity"));
});

test("does not interpolate a quote in the identity identifiers", () => {
  const { query, query_params } = buildExternalIdBackfillMutation(
    "default.key_verifications_per_month_v2",
    { ...row, identity_id: "id' OR 1=1 --" },
    "customer-1",
  );

  assert.equal(query_params.identityId, "id' OR 1=1 --");
  assert.ok(!query.includes("OR 1=1"));
  assert.ok(query.includes("identity_id = {identityId: String}"));
});

test("accepts a plain table name and rejects anything else", () => {
  const { query } = buildExternalIdBackfillMutation(
    "default.key_verifications_per_month_v2",
    row,
    "customer-1",
  );
  assert.ok(query.includes("UPDATE default.key_verifications_per_month_v2"));

  assert.throws(() =>
    buildExternalIdBackfillMutation(
      "default.key_verifications_per_month_v2 SET external_id = '' --",
      row,
      "customer-1",
    ),
  );
});
