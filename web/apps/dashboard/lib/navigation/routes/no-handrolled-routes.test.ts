import { readFileSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

/**
 * Regression guard for the workspace-scoped areas that link through the typed
 * `routes` builders. It fails if a hand-rolled internal path (href="/...",
 * router.push("/..."), redirect("/...")) is introduced, so reviewers catch it
 * instead of shipping an untyped url. New builder areas should be added here.
 *
 * Scope and limits: only `[workspaceSlug]/<area>` dirs are scanned, so routes
 * outside that tree (auth, /new) are covered by their own builders, not this
 * guard. Detection is nav-context only (href / push / replace / redirect), so
 * it won't catch a path stored in a variable and used indirectly. A line with
 * a `route-guard-ignore` comment is skipped for deliberate exceptions.
 */
const AREAS = [
  "apis",
  "projects",
  "ratelimits",
  "settings",
  "authorization",
  "identities",
  "audit",
  "logs",
] as const;

const WORKSPACE_DIR = join(
  dirname(fileURLToPath(import.meta.url)),
  "../../../app/(app)/[workspaceSlug]",
);

// An href attribute or a push/replace/redirect call whose argument is a string
// or template literal opening with an absolute path (`/`). Quote can be ' " or
// a backtick, optionally wrapped in JSX braces for href.
const HAND_ROLLED = /href=\{?\s*["'`]\/|\b(?:push|replace|redirect)\(\s*["'`]\//;
const EXTERNAL = /https?:\/\/|mailto:/;
const COMMENT = /^\s*(?:\/\/|\*|\/\*)/;
const IGNORE = /route-guard-ignore/;

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      return sourceFiles(path);
    }
    return /\.tsx?$/.test(entry.name) ? [path] : [];
  });
}

function violations(dir: string): string[] {
  return sourceFiles(dir).flatMap((file) => {
    const lines = readFileSync(file, "utf8").split("\n");
    return lines.flatMap((line, index) => {
      // The ignore marker is honored on the offending line or the line above
      // it, so it can sit in a leading comment with a reason.
      const exempt = IGNORE.test(line) || (index > 0 && IGNORE.test(lines[index - 1]));
      if (COMMENT.test(line) || EXTERNAL.test(line) || exempt) {
        return [];
      }
      return HAND_ROLLED.test(line) ? [`${file}:${index + 1}: ${line.trim()}`] : [];
    });
  });
}

describe("no hand-rolled routes in migrated areas", () => {
  it.each(AREAS)("%s links through the routes builders", (area) => {
    expect(violations(join(WORKSPACE_DIR, area))).toEqual([]);
  });
});
