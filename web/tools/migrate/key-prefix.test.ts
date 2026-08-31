import assert from "node:assert/strict";
import test from "node:test";
import { extractLegacyKeyPrefix } from "./key-prefix";

test("extracts a prefix from a legacy key start", () => {
  assert.equal(extractLegacyKeyPrefix("prod_abcd"), "prod");
});

test("extracts a prefix that contains underscores", () => {
  assert.equal(extractLegacyKeyPrefix("prod_sk_abcd"), "prod_sk");
});

test("returns null for a key start without a prefix", () => {
  assert.equal(extractLegacyKeyPrefix("abcd"), null);
});

test("returns null when the separator is in the wrong position", () => {
  assert.equal(extractLegacyKeyPrefix("prod_ab_cde"), null);
});

test("returns null when the random characters are not Base58", () => {
  assert.equal(extractLegacyKeyPrefix("prod_abc0"), null);
  assert.equal(extractLegacyKeyPrefix("prod_abcO"), null);
  assert.equal(extractLegacyKeyPrefix("prod_abcI"), null);
  assert.equal(extractLegacyKeyPrefix("prod_abcl"), null);
});
