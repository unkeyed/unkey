import { Hono } from "hono";
import { describe, expect, test } from "vitest";
import { UnkeyContext, unkey } from "./index";

const key = "hono_3ZHg8eyRMts88vxy5uvWLb8S";

describe("No custom Config", () => {
  describe("happy path", () => {
    const app = new Hono<{ Variables: { unkey: UnkeyContext } }>();

    app.use("/*", unkey());

    app.get("/", (c) => c.json(c.get("unkey")));

    test("Should be hello message", async () => {
      const res = await app.request("http://localhost/", {
        headers: {
          Authorization: `Bearer ${key}`,
        },
      });
      expect(res.status).toBe(200);
      expect(await res.json()).toMatchObject({ valid: true });
    });
  });

  describe("No Authorization header", () => {
    const app = new Hono<{ Variables: { unkey: UnkeyContext } }>();

    app.use("/*", unkey());

    app.get("/", (c) => c.json(c.get("unkey")));

    test("should be unauthorized", async () => {
      const res = await app.request("http://localhost/");
      expect(res.status).toBe(401);
      expect(await res.json()).toMatchObject({
        error: "unauthorized",
      });
    });
  });
});

describe("With custom key getter", () => {
  describe("No Authorization header", () => {
    const app = new Hono<{ Variables: { unkey: UnkeyContext } }>();

    app.use(
      "/*",
      unkey({
        getKey: (c) => {
          return c.text("oh well");
        },
      }),
    );

    app.get("/", (c) => c.json(c.get("unkey")));

    test("should be called with response", async () => {
      const res = await app.request("http://localhost/");
      expect(res.status).toBe(200);
      expect(await res.text()).toEqual("oh well");
    });
  });
});

describe("With custom invald handler", () => {
  describe("No Authorization header", () => {
    const app = new Hono<{ Variables: { unkey: UnkeyContext } }>();

    let calledWith: UnkeyContext | null = null;
    app.use(
      "/*",
      unkey({
        handleInvalidKey: (c, result) => {
          calledWith = result;
          return c.text("oh well");
        },
      }),
    );

    app.get("/", (c) => c.json(c.get("unkey")));

    test("should be called with response", async () => {
      const res = await app.request("http://localhost/", {
        headers: {
          Authorization: "Bearer notakey",
        },
      });
      expect(res.status).toBe(200);
      expect(await res.text()).toEqual("oh well");
      expect(calledWith).toMatchObject({
        valid: false,
        code: "NOT_FOUND",
      });
    });
  });
});
