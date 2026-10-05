import { describe, expect, test } from "vitest";
import { createRootKeyInput, updateRootKeyPermissionsInput } from "./root-key-permissions-input";

const legacyPermission = "project.proj_12345678.read_project";
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
  test("accepts legacy permissions", () => {
    expect(parse([legacyPermission])).toBe(true);
  });

  test("rejects URN permissions", () => {
    expect(parse([urnPermission])).toBe(false);
  });
});
