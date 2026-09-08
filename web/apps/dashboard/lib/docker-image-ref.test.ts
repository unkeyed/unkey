import { describe, expect, it } from "vitest";
import { sanitizeImageRef, validateImageRef } from "./docker-image-ref";

const digest = `sha256:${"a".repeat(64)}`;

describe("validateImageRef", () => {
  it("accepts references registries accept", () => {
    const valid = [
      "mysql",
      "kebap",
      "library/redis:7",
      "ghcr.io/acme/mcp-server:v1.4.2",
      "docker.io/library/mysql:8.0.36",
      "index.docker.io/library/mysql",
      // Whether a host serves images is not knowable here, so any well-formed
      // reference passes and an unservable one fails at pull time.
      "gitlab.com/acme/api",
      "github.com/acme/api",
      "localhost:5000/acme/api:dev",
      "registry.internal:5000/team/api",
      "[::1]:5000/acme/api",
      "127.0.0.1:5000/api",
      "UPPER.EXAMPLE.COM/acme/api",
      "acme/api:_v1",
      "a__b/c",
      "a".repeat(247),
      `acme/api@sha384:${"a".repeat(96)}`,
      `acme/api:${"t".repeat(128)}`,
      "quay.io/acme/api.v2_beta-1:2024.01.01",
      `ghcr.io/acme/api@${digest}`,
      `ghcr.io/acme/api:v1@${digest}`,
    ];
    for (const ref of valid) {
      expect(validateImageRef(ref), ref).toMatchObject({ ok: true });
    }
  });

  it("splits the reference into its parts", () => {
    expect(validateImageRef("ghcr.io/acme/api:v1")).toMatchObject({
      ok: true,
      parts: { domain: "ghcr.io", path: "acme/api", tag: "v1", digest: undefined },
    });
    expect(validateImageRef("mysql")).toMatchObject({
      ok: true,
      parts: { domain: undefined, path: "mysql" },
    });
    expect(validateImageRef("localhost:5000/api")).toMatchObject({
      ok: true,
      parts: { domain: "localhost:5000", path: "api" },
    });
    expect(validateImageRef(`ghcr.io/acme/api:v1@${digest}`)).toMatchObject({
      ok: true,
      parts: { domain: "ghcr.io", path: "acme/api", tag: "v1", digest },
    });
  });

  it("warns about mutable tags without blocking", () => {
    expect(validateImageRef("mysql")).toMatchObject({
      ok: true,
      warning: expect.stringContaining(":latest"),
    });
    expect(validateImageRef("mysql:latest")).toMatchObject({
      ok: true,
      warning: expect.stringContaining("mutable"),
    });
    expect(validateImageRef(`mysql:latest@${digest}`)).not.toHaveProperty("warning");
  });

  it("rejects malformed references", () => {
    const invalid: [string, string][] = [
      ["", "required"],
      ["   ", "required"],
      ["my image:v1", "spaces"],
      ["https://ghcr.io/acme/api", "protocol"],
      ["Acme/Api:v1", "must be lowercase"],
      ["ghcr.io/Acme/api", "must be lowercase"],
      ["mysql:", "empty"],
      ["mysql:-bad", "tag can only"],
      [`mysql:v1@${"x".repeat(20)}`, "digest"],
      ["mysql@sha256:abc123", "hex characters"],
      [`mysql@sha256:${"A".repeat(64)}`, "lowercase hex"],
      [`mysql@sha512:${"a".repeat(64)}`, "128 hex characters"],
      [`mysql@${digest}@${digest}`, "one @"],
      ["ghcr.io//acme/api", "empty path segment"],
      ["hub.docker.com/_/nginx", "can only contain lowercase"],
      ["ghcr.io-/acme/api", "not a valid registry host"],
      ["acme/-api", "can only contain lowercase"],
      // Measured after the `library/` expansion, so 248 is already one over.
      ["a".repeat(248), "longer than 255"],
      ["a".repeat(256), "longer than 255"],
      ["a__ _b/c", "spaces"],
      ["acme/a___b", "can only contain lowercase"],
      ["acme/api-", "can only contain lowercase"],
      [":v1", "image name is missing"],
      [`acme/api@blake3:${"a".repeat(32)}`, "cannot be pulled"],
      [`acme/api@sha256+b64:${"a".repeat(32)}`, "cannot be pulled"],
      [`mysql:${"a".repeat(129)}`, "longer than 128"],
      [`ghcr.io/acme/${"a".repeat(100)}:${"v".repeat(128)}@${digest}`, "longer than 256"],
    ];
    for (const [ref, fragment] of invalid) {
      const result = validateImageRef(ref);
      expect(result.ok, ref).toBe(false);
      if (!result.ok) {
        expect(result.error, ref).toContain(fragment);
      }
    }
  });
});

describe("sanitizeImageRef", () => {
  it("strips what a reference cannot legally contain", () => {
    expect(sanitizeImageRef("  ghcr.io/acme/api:v1  ")).toBe("ghcr.io/acme/api:v1");
    expect(sanitizeImageRef('"ghcr.io/acme/api:v1"')).toBe("ghcr.io/acme/api:v1");
    expect(sanitizeImageRef("https://ghcr.io/acme/api:v1")).toBe("ghcr.io/acme/api:v1");
  });

  it("lowercases the name but not the tag or digest", () => {
    expect(sanitizeImageRef("ghcr.io/Acme/Api")).toBe("ghcr.io/acme/api");
    expect(sanitizeImageRef("Acme/Api:V1.2-RC1")).toBe("acme/api:V1.2-RC1");
    expect(sanitizeImageRef("UPPER.EXAMPLE.COM/Acme/api")).toBe("upper.example.com/acme/api");
    expect(sanitizeImageRef(`Acme/Api@${digest}`)).toBe(`acme/api@${digest}`);
    expect(sanitizeImageRef(`Acme/Api:V1@${digest}`)).toBe(`acme/api:V1@${digest}`);
  });

  it("leaves a valid reference untouched", () => {
    expect(sanitizeImageRef("localhost:5000/acme/api:dev")).toBe("localhost:5000/acme/api:dev");
    expect(sanitizeImageRef("[::1]:5000/acme/api:Dev")).toBe("[::1]:5000/acme/api:Dev");
  });

  it("leaves a pasted command line for validation to reject", () => {
    for (const command of ["docker pull nginx:1.27", "$ docker pull nginx"]) {
      expect(validateImageRef(sanitizeImageRef(command))).toMatchObject({
        ok: false,
        error: expect.stringContaining("spaces"),
      });
    }
  });
});
