import { fileURLToPath } from "node:url";
import { $fetch, fetch, setup } from "@nuxt/test-utils";
import { config } from "dotenv";
import { describe, expect, it } from "vitest";

config({
  path: fileURLToPath(new URL("../.env", import.meta.url)),
});

await setup({
  rootDir: fileURLToPath(new URL("../playground", import.meta.url)),
});

describe("basic behaviour", () => {
  it("should set `unkey` on context with (invalid) authenticated request", async () => {
    const res = await fetch("/api/context", {
      headers: {
        Authorization: "Bearer 123",
      },
    });
    expect(await res.json()).toMatchInlineSnapshot(`
      {
        "unkey": {
          "code": "NOT_FOUND",
          "valid": false,
        },
      }
    `);
  });

  // TODO: use mock API when we have one
  it("should set `unkey` on context with (valid) authenticated request", async () => {
    const data = await $fetch("/api/context", {
      headers: {
        Authorization: `Bearer ${process.env.NUXT_TEST_KEY}`,
      },
    });
    expect(data).toMatchInlineSnapshot(`
      {
        "unkey": {
          "code": "NOT_FOUND",
          "valid": false,
        },
      }
    `);
  });

  it("should allow using `useUnkey()` helper", async () => {
    const data = await $fetch("/api/helper");
    expect(data).toMatchInlineSnapshot(`
      {
        "baseUrl": "https://api.unkey.dev",
      }
    `);
  });
});
