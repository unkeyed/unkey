import { describe, expect, it } from "vitest";
import { changedIpAllowlist, rootKeySchema, splitIpAllowlist } from "./schema";

describe("root key IP allowlist", () => {
  it("omits unchanged allowlists so unrelated saves cannot overwrite a newer policy", () => {
    expect(changedIpAllowlist("", [])).toBeUndefined();
    expect(changedIpAllowlist(" 192.0.2.0/24\n\n", ["192.0.2.0/24"])).toBeUndefined();
    expect(changedIpAllowlist("", ["192.0.2.0/24"])).toEqual([]);
    expect(changedIpAllowlist("192.0.2.1\n2001:db8::1", ["192.0.2.0/24"])).toEqual([
      "192.0.2.1",
      "2001:db8::1",
    ]);
  });

  it("accepts addresses, ranges, and an empty allowlist", () => {
    for (const ipAllowlist of [
      "",
      " \n ",
      "192.0.2.1\n198.51.100.0/24\n2001:db8::1\n2001:db8::/32",
    ]) {
      expect(rootKeySchema.partial().safeParse({ ipAllowlist }).success).toBe(true);
    }
    expect(splitIpAllowlist(" 192.0.2.1 \n\n 2001:db8::/32 ")).toEqual([
      "192.0.2.1",
      "2001:db8::/32",
    ]);
    expect(splitIpAllowlist(" \n ")).toEqual([]);
  });

  it("rejects malformed addresses, ranges, and oversized lists", () => {
    for (const ipAllowlist of [
      "not-an-ip",
      "192.0.2.1/33",
      "2001:db8::/129",
      "192.0.2.1\ninvalid",
      Array.from({ length: 101 }, () => "192.0.2.1").join("\n"),
    ]) {
      expect(rootKeySchema.partial().safeParse({ ipAllowlist }).success).toBe(false);
    }
  });
});
