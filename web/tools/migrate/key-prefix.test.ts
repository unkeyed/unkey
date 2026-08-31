import assert from "node:assert/strict";
import test from "node:test";
import { extractLegacyKeyMetadata } from "./key-prefix";

test("separates a prefix from a legacy key start", () => {
  assert.deepEqual(extractLegacyKeyMetadata("prod_abcd"), {
    prefix: "prod",
    start: "abcd",
  });
});

test("separates a prefix that contains underscores", () => {
  assert.deepEqual(extractLegacyKeyMetadata("prod_sk_abcd"), {
    prefix: "prod_sk",
    start: "abcd",
  });
});

test("returns null for a key start without a prefix", () => {
  assert.equal(extractLegacyKeyMetadata("abcd"), null);
});

test("returns null when the separator is in the wrong position", () => {
  assert.equal(extractLegacyKeyMetadata("prod_ab_cde"), null);
});

test("returns null when the random characters are not Base58", () => {
  assert.equal(extractLegacyKeyMetadata("prod_abc0"), null);
  assert.equal(extractLegacyKeyMetadata("prod_abcO"), null);
  assert.equal(extractLegacyKeyMetadata("prod_abcI"), null);
  assert.equal(extractLegacyKeyMetadata("prod_abcl"), null);
});
