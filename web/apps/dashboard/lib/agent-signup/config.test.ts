import { describe, expect, it } from "vitest";
import { agentSignupEnv, authKitIssuer } from "./config";

const enabled = {
  AUTH_PROVIDER: "workos",
  WORKOS_API_KEY: "sk_test",
  WORKOS_CLIENT_ID: "client_123",
  WORKOS_AUTHKIT_DOMAIN: "auth.example.com",
  WORKOS_AGENT_AUDIENCE: "client_123",
};

describe("authKitIssuer", () => {
  it("accepts a host or an https origin", () => {
    expect(authKitIssuer("auth.example.com")).toBe("https://auth.example.com");
    expect(authKitIssuer("https://auth.example.com/")).toBe("https://auth.example.com");
  });

  it("rejects http and paths", () => {
    expect(authKitIssuer("http://auth.example.com")).toBeNull();
    expect(authKitIssuer("https://auth.example.com/agent")).toBeNull();
  });
});

describe("agentSignupEnv", () => {
  it("is disabled until AuthKit domain and audience are set", () => {
    expect(agentSignupEnv({ AUTH_PROVIDER: "local" })).toBeNull();
    expect(agentSignupEnv({ ...enabled, WORKOS_AUTHKIT_DOMAIN: "" })).toBeNull();
    expect(agentSignupEnv(enabled)).toMatchObject({
      issuer: "https://auth.example.com",
      audience: "client_123",
      apiBase: "https://api.workos.com",
      jwksUrl: "https://auth.example.com/oauth2/jwks",
    });
  });
});
