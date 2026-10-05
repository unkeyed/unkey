import { describe, expect, test } from "vitest";
import { createRootKeyInput, updateRootKeyPermissionsInput } from "./root-key-permissions-input";

const malformedPermission = "not-a-permission";
const urnPermission = "unkey:v1:ws_12345678:projects/*#read_project";

describe.each([
  {
    name: "create root key",
    parse: (permissions: string[]) =>
      createRootKeyInput.safeParse({ name: "Production", permissions }).success,
  },
  {
    name: "update root key permissions",
    parse: (permissions: string[]) =>
      updateRootKeyPermissionsInput.safeParse({ keyId: "key_12345678", permissions }).success,
  },
])("$name input", ({ parse }) => {
  test("accepts URN permissions", () => {
    expect(parse([urnPermission])).toBe(true);
  });

  test("rejects malformed permissions", () => {
    expect(parse([malformedPermission])).toBe(false);
  });
});
