import { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import { assert, describe, expect, it } from "vitest";
import { defaultRatelimitIdentifier } from "../../middleware/util";

describe("defaultRatelimitIdentifier", () => {
  it("should return correct identifier", () => {
    const request = new Request("http://localhost:8910/api/user");
    const req = new MiddlewareRequest(request);
    assert.equal(defaultRatelimitIdentifier(req), "192.168.1.1");
  });

  it("when authenticated, should return correct identifier", () => {
    const request = new Request("http://localhost:8910/api/user");
    const req = new MiddlewareRequest(request);
    req.serverAuthContext.set({
      isAuthenticated: true,
      currentUser: { id: 1 },
    });
    assert.equal(defaultRatelimitIdentifier(req), "eyJpZCI6MX0=");
  });

  it("when not authenticated, should return correct identifier", () => {
    const request = new Request("http://localhost:8910/api/user");
    const req = new MiddlewareRequest(request);
    req.serverAuthContext.set({
      isAuthenticated: false,
      currentUser: { id: 1 },
    });
    assert.equal(defaultRatelimitIdentifier(req), "192.168.1.1");
  });
});
