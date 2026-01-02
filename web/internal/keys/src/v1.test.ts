import { expect, test } from "vitest";
import { KeyV1 } from "./v1";

test("create v1 key", () => {
  const key = new KeyV1({ byteLength: 16 });
  expect(key.toString()).toMatch(/^[a-zA-Z0-9]+$/);
});

test("unmarshal", () => {
  const key = new KeyV1({ prefix: "prfx", byteLength: 16 });
  const key2 = KeyV1.fromString(key.toString());
  expect(key2.toString()).toEqual(key.toString());
  expect(key2.prefix).toEqual("prfx");
  expect(key2.random).toEqual(key.random);
});
