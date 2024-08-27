import { describe, expect, test } from "vitest";
import { Unkey } from "./client";


describe("client", () => {
  test("fetch can encode the params without throwing", async () => {
    const unkey = new Unkey({ token: 'rawr'});
    expect(() => {
      unkey.apis.listKeys({
        apiId: 'meow',
        cursor: undefined,
      });
    }).not.toThrow();
  });
});
