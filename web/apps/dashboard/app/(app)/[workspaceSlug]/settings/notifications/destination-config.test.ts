import { describe, expect, it } from "vitest";
import {
  destinationUrlProblem,
  generateSigningSecret,
  headerNameProblem,
  secretHint,
  urlHint,
} from "./destination-config";

describe("destinationUrlProblem", () => {
  it("accepts channel-specific webhook URLs", () => {
    expect(destinationUrlProblem("discord", "https://discord.com/api/webhooks/1/abc")).toBeNull();
    expect(destinationUrlProblem("slack", "https://hooks.slack.com/services/T/B/x")).toBeNull();
    expect(destinationUrlProblem("webhook", "https://example.com/hook")).toBeNull();
  });

  it("rejects non-https and wrong hosts", () => {
    expect(destinationUrlProblem("webhook", "http://example.com/hook")).toMatch(/https/);
    expect(destinationUrlProblem("slack", "https://example.com/services/x")).toMatch(/Slack/);
    expect(destinationUrlProblem("discord", "https://discord.com/channels/1")).toMatch(/Discord/);
    expect(destinationUrlProblem("webhook", "not a url")).toMatch(/full URL/);
  });
});

describe("headerNameProblem", () => {
  it("accepts tokens and rejects spaces", () => {
    expect(headerNameProblem("X-Routing-Key")).toBeNull();
    expect(headerNameProblem("Bad Header")).not.toBeNull();
    expect(headerNameProblem("")).not.toBeNull();
  });
});

describe("hints", () => {
  it("keeps only the host, first path segment, and last four characters", () => {
    expect(urlHint("https://hooks.slack.com/services/T000/B000/abcd9kLm")).toBe(
      "hooks.slack.com/services/…9kLm",
    );
    expect(urlHint("https://example.com/")).toBe("example.com");
  });

  it("generates whsec secrets and hints their tail", () => {
    const secret = generateSigningSecret();
    expect(secret).toMatch(/^whsec_[0-9a-f]{48}$/);
    expect(secretHint(secret)).toBe(`whsec_…${secret.slice(-4)}`);
  });
});
