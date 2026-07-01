import type { ErrorEvent, EventHint } from "@sentry/nextjs";
import { describe, expect, it } from "vitest";
import { createErrorFilter } from "./error-filter";

describe("createErrorFilter", () => {
  it("redacts secret tRPC input before forwarding unexpected errors", () => {
    const secret = "unkey_secret_plaintext_value";
    const event = {
      type: undefined,
      contexts: {
        trpc: {
          path: "share.create",
          input: {
            secret,
            variables: [{ value: "env_secret_value" }],
            safe: "kept",
          },
        },
      },
    } as unknown as ErrorEvent;

    const filter = createErrorFilter({ logFilteredErrors: false });
    if (!filter) {
      throw new Error("Expected error filter to be defined");
    }

    const result = filter(event, {
      originalException: new Error("vault unavailable"),
    } as EventHint);

    expect(result).toBe(event);
    expect(JSON.stringify(event.contexts?.trpc)).not.toContain(secret);
    expect(event.contexts?.trpc?.input).toEqual({
      secret: "[REDACTED]",
      variables: [{ value: "[REDACTED]" }],
      safe: "kept",
    });
  });
});
