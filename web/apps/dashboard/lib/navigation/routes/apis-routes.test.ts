import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";
const apiId = "api_123";
const keyAuthId = "ks_456";
const keyId = "key_789";

describe("api-scoped paths", () => {
  it("builds the list and api base paths", () => {
    expect(routes.apis.list({ workspaceSlug: ws })).toBe("/acme/apis");
    expect(routes.apis.detail({ workspaceSlug: ws, apiId })).toBe("/acme/apis/api_123");
  });

  it("appends the new query when flagged", () => {
    expect(routes.apis.list({ workspaceSlug: ws, new: true })).toBe("/acme/apis?new=true");
  });

  it("builds the settings path", () => {
    expect(routes.apis.settings({ workspaceSlug: ws, apiId })).toBe("/acme/apis/api_123/settings");
  });
});

describe("key-scoped paths", () => {
  it("scopes to a keyspace", () => {
    expect(routes.apis.keys.list({ workspaceSlug: ws, apiId, keyAuthId })).toBe(
      "/acme/apis/api_123/keys/ks_456",
    );
  });

  it("scopes to a single key", () => {
    expect(routes.apis.keys.detail({ workspaceSlug: ws, apiId, keyAuthId, keyId })).toBe(
      "/acme/apis/api_123/keys/ks_456/key_789",
    );
  });
});

/**
 * Guard against reintroducing hand-rolled /apis urls. Biome has no custom-rule
 * support, so this regex walk over the app tree is the enforcement: every
 * navigation into the apis area must go through routes.apis.* so a renamed
 * segment or wrong param fails to compile instead of 404ing at runtime.
 *
 * A path literal is any quoted/backtick string that starts with `/`. We flag it
 * only when `apis` appears as a NON-first segment (a `/<slug>/apis...` shape) —
 * that is a workspace-scoped route that must use the builder. A bare `/apis`
 * (apis first) has no slug in scope (auth/join/onboarding redirects rely on
 * middleware to inject it) and is not an AppRoutes member, so buildRoute cannot
 * express it; those pass. Import specifiers (`@/...`, `./...`) start with
 * another char and never match.
 */
const SCAN_DIRS = ["app", "components", "hooks", "lib"].map((d) => join(process.cwd(), d));
const ROUTES_DIR = join(process.cwd(), "lib/navigation/routes");
const PATH_LITERAL = /([`"'])(\/[^`"'\n]*?)\1/g;
const APIS_SEGMENT = /\/[^/`"'\n]+\/apis(\/|\?|$)/;

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) {
      return full.startsWith(ROUTES_DIR) ? [] : walk(full);
    }
    return full.endsWith(".ts") || full.endsWith(".tsx") ? [full] : [];
  });
}

describe("no hand-rolled /apis urls", () => {
  it("routes every workspace-scoped apis navigation through routes.apis.*", () => {
    const violations: string[] = [];

    for (const file of SCAN_DIRS.flatMap(walk)) {
      const rel = relative(process.cwd(), file);
      if (rel.endsWith(".test.ts") || rel.endsWith(".test.tsx")) {
        continue;
      }
      const lines = readFileSync(file, "utf8").split("\n");
      lines.forEach((line, i) => {
        for (const [, , inner] of line.matchAll(PATH_LITERAL)) {
          if (APIS_SEGMENT.test(inner)) {
            violations.push(`${rel}:${i + 1} — ${inner.trim()}`);
          }
        }
      });
    }

    expect(
      violations,
      `Hand-rolled /apis urls found. Use routes.apis.* from "@/lib/navigation/routes":\n${violations.join("\n")}`,
    ).toEqual([]);
  });
});
