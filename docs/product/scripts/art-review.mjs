import assert from "node:assert/strict";

// A new or renamed card needs a human quality decision before it can publish art.
export function validateArtReview(entries, expectedIds) {
  assert.ok(Array.isArray(entries), "Artwork review must be an array");
  const expected = new Set(expectedIds);
  const reviewed = new Map();
  for (const entry of entries) {
    assert.ok(expected.has(entry.id), `Unknown artwork review: ${entry.id}`);
    assert.ok(!reviewed.has(entry.id), `Duplicate artwork review: ${entry.id}`);
    assert.ok(["keep", "remove"].includes(entry.decision), `Invalid decision: ${entry.id}`);
    assert.ok(
      typeof entry.reason === "string" && entry.reason.trim().length > 0,
      `Missing reason: ${entry.id}`,
    );
    reviewed.set(entry.id, entry);
  }
  for (const id of expected) assert.ok(reviewed.has(id), `Artwork needs quality review: ${id}`);
  return reviewed;
}
