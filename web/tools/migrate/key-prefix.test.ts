import assert from "node:assert/strict";
import test from "node:test";
import { splitLegacyKeyStart } from "./key-prefix";

test("separates a prefix from a legacy key start", () => {
  assert.deepEqual(splitLegacyKeyStart("prod_abcd"), {
    prefix: "prod",
    start: "abcd",
  });
});

test("separates a prefix that contains underscores", () => {
  assert.deepEqual(splitLegacyKeyStart("prod_sk_abcd"), {
    prefix: "prod_sk",
    start: "abcd",
  });
});

test("returns null for a key start without a prefix", () => {
  assert.equal(splitLegacyKeyStart("abcd"), null);
});

test("returns null when the separator is in the wrong position", () => {
  assert.equal(splitLegacyKeyStart("prod_ab_cde"), null);
});

test("keeps the existing key start characters", () => {
  assert.deepEqual(splitLegacyKeyStart("prod_abc0"), {
    prefix: "prod",
    start: "abc0",
  });
});
