import type { ErrorEvent, EventHint } from "@sentry/nextjs";
import { describe, expect, it } from "vitest";
import { createErrorFilter } from "./error-filter";

function makeFilter() {
  const filter = createErrorFilter({ logFilteredErrors: false });
  if (!filter) {
    throw new Error("Expected error filter to be defined");
  }
  return filter;
}

describe("createErrorFilter", () => {
  it("redacts secret tRPC input before forwarding unexpected errors", () => {
    const secret = "unkey_secret_plaintext_value";
    const event: ErrorEvent = {
      type: undefined,
      contexts: {
        trpc: {
          procedure_path: "share.create",
          input: {
            secret,
            variables: [{ value: "env_secret_value" }],
            safe: "kept",
          },
        },
      },
    };
    const hint: EventHint = { originalException: new Error("vault unavailable") };

    const result = makeFilter()(event, hint);

    expect(result).toBe(event);
    expect(JSON.stringify(event.contexts?.trpc)).not.toContain(secret);
    expect(event.contexts?.trpc?.input).toEqual({
      secret: "[REDACTED]",
      variables: [{ value: "[REDACTED]" }],
      safe: "kept",
    });
  });

  it("redacts credential-bearing keys case-insensitively and token-like values under any key", () => {
    const totpCode = "123456";
    const oauthCode = "gho_abcdefghijklmnopqrstuvwxyz012345";
    const opaqueToken = "unkey_3ZjJ2Zm9uZGF0aW9uX3Rlc3Q";
    const event: ErrorEvent = {
      type: undefined,
      contexts: {
        trpc: {
          procedure_path: "user.verifyMfaEnrollment",
          input: {
            code: totpCode,
            Token: oauthCode,
            note: `see ${opaqueToken} for details`,
            safe: "kept",
          },
        },
      },
    };
    const hint: EventHint = { originalException: new Error("mfa backend down") };

    const result = makeFilter()(event, hint);

    expect(result).toBe(event);
    const serialized = JSON.stringify(event);
    expect(serialized).not.toContain(totpCode);
    expect(serialized).not.toContain(oauthCode);
    expect(serialized).not.toContain(opaqueToken);
    expect(event.contexts?.trpc?.input).toEqual({
      code: "[REDACTED]",
      Token: "[REDACTED]",
      note: "see [REDACTED] for details",
      safe: "kept",
    });
  });

  it("redacts the entire input for procedures whose input is a credential", () => {
    const shareId = "still_valid_one_time_share_id";
    const event: ErrorEvent = {
      type: undefined,
      contexts: {
        trpc: {
          procedure_path: "share.reveal",
          procedure_type: "mutation",
          input: { id: shareId },
        },
      },
    };
    const hint: EventHint = { originalException: new Error("vault unavailable") };

    const result = makeFilter()(event, hint);

    expect(result).toBe(event);
    expect(event.contexts?.trpc?.input).toBe("[REDACTED]");
    expect(JSON.stringify(event)).not.toContain(shareId);
  });
});
