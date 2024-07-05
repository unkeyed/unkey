import { SELF, createExecutionContext, env, waitOnExecutionContext } from "cloudflare:test";
import { describe, expect, it } from "vitest";
import worker from "../src";

describe("Hello World worker", () => {
  it("responds with Hello World! (unit style)", async () => {
    const request = new Request("http://example.com");
    // Create an empty context to pass to `worker.fetch()`.
    const ctx = createExecutionContext();
    const response = await worker.fetch(request, env, ctx);
    // Wait for all `Promise`s passed to `ctx.waitUntil()` to settle before running test assertions
    await waitOnExecutionContext(ctx);
    expect(await response.text()).toMatchInlineSnapshot(`"Hello World!"`);
  });

  it("responds with Hello World! (integration style)", async () => {
    const response = await SELF.fetch(request, env, ctx);
    expect(await response.text()).toMatchInlineSnapshot(`"Hello World!"`);
  });
});
