import { describe, expect, it } from "vitest";
import { oauthClientOf } from "./oauth-client";

const oauthClientId = "https://client.example.com/oauth/client.json";

describe("oauthClientOf", () => {
  it("treats a login without a client as a CLI login", () => {
    expect(oauthClientOf({ deviceName: "Test laptop" })).toBeNull();
  });

  it("pairs the self-reported name with the client's host", () => {
    expect(oauthClientOf({ oauthClientId, deviceName: "Example MCP" })).toEqual({
      name: "Example MCP",
      host: "client.example.com",
    });
  });

  it("shows no host for Unkey's hosted MCP server", () => {
    expect(
      oauthClientOf({ oauthClientId, oauthClientFirstParty: true, deviceName: "Unkey MCP" }),
    ).toEqual({ name: "Unkey MCP", host: null });
  });

  it("keeps the host for a third party that calls itself Unkey MCP", () => {
    expect(
      oauthClientOf({ oauthClientId, oauthClientFirstParty: false, deviceName: "Unkey MCP" }),
    ).toEqual({ name: "Unkey MCP", host: "client.example.com" });
  });

  it("names the client by its host when it reports no name", () => {
    expect(oauthClientOf({ oauthClientId })).toEqual({
      name: "client.example.com",
      host: "client.example.com",
    });
  });
});
