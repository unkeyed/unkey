import { describe, expect, test } from "vitest";
import { Unkey } from "./client";

describe("client", () => {
  test("fetch can encode the params without throwing", async () => {
    const unkey = new Unkey({ token: "rawr" });
    expect(() => {
      unkey.apis.listKeys({
        apiId: "meow",
        cursor: undefined,
      });
    }).not.toThrow();
  });

  test("errors are correctly passed through to the caller", async () => {
    const unkey = new Unkey({ rootKey: "wrong key" });
    const res = await unkey.keys.create({
      apiId: "",
    });

    expect(res.error).toBeDefined();
    expect(res.error!.code).toEqual("UNAUTHORIZED");
    expect(res.error!.docs).toEqual(
      "https://unkey.dev/docs/api-reference/errors/code/UNAUTHORIZED",
    );
    expect(res.error!.message).toEqual("key not found");
    expect(res.error!.requestId).toBeDefined();
  });
});
