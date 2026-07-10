import { ForbiddenErrorResponse } from "@unkey/api/models/errors";
import { describe, expect, it } from "vitest";
import { getErrorMessage } from "./unkey-client";

describe("getErrorMessage", () => {
  it("returns the SDK error detail", () => {
    const error = new ForbiddenErrorResponse(
      {
        error: {
          detail: "Missing one of these permissions: api.*.update_key",
          status: 403,
          title: "Insufficient Permissions",
          type: "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions",
        },
        meta: {
          requestId: "req_123",
        },
      },
      {
        body: "",
        request: new Request("https://api.unkey.com/v2/keys.updateKey"),
        response: new Response(null, { status: 403 }),
      },
    );

    expect(getErrorMessage(error)).toBe("Missing one of these permissions: api.*.update_key");
  });

  it("returns the fallback for non-SDK errors", () => {
    expect(getErrorMessage(new Error("Network request failed"), "Try again")).toBe("Try again");
  });
});
