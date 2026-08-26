import * as errors from "@unkey/api/models/errors";
import { beforeEach, describe, expect, it } from "vitest";
import { withIdempotencyKey } from "./idempotency";

function badRequest() {
  return new errors.BadRequestErrorResponse(
    {
      meta: { requestId: "req_KEBAP" },
      error: {
        title: "Bad Request",
        detail: "This Idempotency-Key belongs to a deployment that ended before its build started.",
        status: 400,
        type: "https://unkey.com/docs/errors/unkey/application/invalid_input",
        errors: [],
      },
    },
    {
      response: new Response(null, { status: 400 }),
      request: new Request("https://api.unkey.com/v2/deployments.createDeployment"),
      body: "",
    },
  );
}

function serverError(status: number) {
  return new errors.UnkeyError("upstream exploded", {
    response: new Response(null, { status }),
    request: new Request("https://api.unkey.com/v2/deployments.createDeployment"),
    body: "",
  });
}

// keyOf runs one request and reports the key it was sent with. When failWith
// is set, the request throws it; withIdempotencyKey rethrows and the test only
// needs the key.
async function keyOf(body: unknown, failWith?: unknown): Promise<string> {
  let sent = "";
  try {
    await withIdempotencyKey(body, async (key) => {
      sent = key;
      if (failWith !== undefined) {
        throw failWith;
      }
      return "ok";
    });
  } catch {
    // expected for failWith runs
  }
  return sent;
}

const deployBody = { app: "KEBAP", image: "nginx:1" };

describe("withIdempotencyKey", () => {
  // Keys persist in sessionStorage; isolate tests from each other.
  beforeEach(() => {
    sessionStorage.clear();
  });

  // A 5xx or a dropped connection may still have created the deployment, and
  // only the unchanged key brings it back instead of duplicating it.
  it("reuses one key across retries of the same body", async () => {
    const first = await keyOf(deployBody, new Error("network down"));
    const second = await keyOf(deployBody, new Error("network down"));

    expect(first).toBeTruthy();
    expect(second).toBe(first);
  });

  // An edited resubmit is a new deployment intent. Keeping the old key would
  // make the server replay the old deployment and silently drop the edit.
  it("rotates when the body changes between attempts", async () => {
    const first = await keyOf(deployBody, new Error("network down"));
    const second = await keyOf({ ...deployBody, image: "nginx:2" }, new Error("network down"));

    expect(second).not.toBe(first);
  });

  // Two in-flight intents must not clobber each other's keys: retrying the
  // first body has to reuse the first key even after a different body ran.
  it("keeps a separate key per in-flight body", async () => {
    const otherBody = { ...deployBody, image: "nginx:2" };

    const first = await keyOf(deployBody, new Error("network down"));
    const other = await keyOf(otherBody, new Error("network down"));
    const retry = await keyOf(deployBody, new Error("network down"));

    expect(other).not.toBe(first);
    expect(retry).toBe(first);
  });

  it("rotates after a settled success", async () => {
    const first = await keyOf(deployBody);
    const second = await keyOf(deployBody);

    expect(second).not.toBe(first);
  });

  // A rejected request created nothing, so the key is free to discard. It
  // also covers a key the API considers spent, which is the one error
  // retrying with the same key can never clear.
  it("rotates when the API rejected the request", async () => {
    const first = await keyOf(deployBody, badRequest());
    const second = await keyOf(deployBody, new Error("network down"));

    expect(second).not.toBe(first);
  });

  it("returns the request's result and rethrows its error", async () => {
    await expect(withIdempotencyKey(deployBody, async () => "KEBAP")).resolves.toBe("KEBAP");

    const boom = new Error("KEBAP exploded");
    await expect(withIdempotencyKey(deployBody, () => Promise.reject(boom))).rejects.toBe(boom);
  });

  // A 5xx is ambiguous: the deployment may exist, so only the same key can
  // find it. Rotating here is exactly the duplicate this exists to stop.
  it("keeps the key after a server error", async () => {
    const first = await keyOf(deployBody, serverError(503));
    const second = await keyOf(deployBody, new Error("network down"));

    expect(second).toBe(first);
  });

  // The failed attempt is often retried after the page reloaded, which only
  // storage survives: an in-memory key would forge a new one and duplicate
  // the deployment the failure may have created.
  it("holds the key in sessionStorage", async () => {
    const key = await keyOf(deployBody, new Error("network down"));

    expect(sessionStorage.getItem(`unkey:deploy-idempotency:${JSON.stringify(deployBody)}`)).toBe(
      key,
    );
  });
});
