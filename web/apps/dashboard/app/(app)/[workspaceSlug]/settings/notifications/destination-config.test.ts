import { describe, expect, it } from "vitest";
import {
  type DestinationFormValues,
  destinationCreateInput,
  destinationFormSchema,
  destinationUpdateInput,
  destinationUrlProblem,
  generateSigningSecret,
  headerNameProblem,
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

describe("write-only destination credentials", () => {
  const destinationId = crypto.randomUUID();
  const values: DestinationFormValues = {
    channel: "webhook",
    name: "On-call",
    url: "",
    headersMode: "keep",
    headers: [],
    enabled: true,
  };

  it("leaves URL, headers and secret out of updates unless replaced", () => {
    expect(destinationFormSchema(true).safeParse(values).success).toBe(true);
    expect(destinationFormSchema(false).safeParse(values).success).toBe(false);
    expect(JSON.parse(JSON.stringify(destinationUpdateInput(destinationId, values, null)))).toEqual(
      { destinationId, name: "On-call", enabled: true },
    );
  });

  it("distinguishes keeping headers from clearing the entire set", () => {
    expect(destinationUpdateInput(destinationId, values, null).headers).toBeUndefined();
    expect(
      destinationUpdateInput(destinationId, { ...values, headersMode: "replace" }, null).headers,
    ).toEqual([]);
  });

  it("ignores abandoned header edits when keeping saved headers", () => {
    const abandoned = { ...values, headers: [{ name: "", value: "x".repeat(2049) }] };
    expect(destinationFormSchema(true).safeParse(abandoned).success).toBe(true);
    expect(destinationUpdateInput(destinationId, abandoned, null).headers).toBeUndefined();
    expect(
      destinationFormSchema(true).safeParse({ ...abandoned, headersMode: "replace" }).success,
    ).toBe(false);
  });

  it("sends complete replacement credentials, never redacted hints", () => {
    const headers = [{ name: " X-Routing-Key ", value: "real-header-value" }];
    const signingSecret = generateSigningSecret();
    const input = destinationUpdateInput(
      destinationId,
      {
        ...values,
        url: " https://example.com/private/path?token=credential ",
        headersMode: "replace",
        headers,
      },
      signingSecret,
    );
    expect(input.url).toBe("https://example.com/private/path?token=credential");
    expect(input.headers).toEqual([{ name: "X-Routing-Key", value: "real-header-value" }]);
    expect(input.signingSecret).toBe(signingSecret);
  });

  it("requires every replacement header value and rejects duplicate names", () => {
    const schema = destinationFormSchema(true);
    expect(
      schema.safeParse({
        ...values,
        headersMode: "replace",
        headers: [{ name: "X-Token", value: "" }],
      }).success,
    ).toBe(false);
    expect(
      schema.safeParse({
        ...values,
        headersMode: "replace",
        headers: [
          { name: "X-Token", value: "first" },
          { name: "x-token", value: "second" },
        ],
      }).success,
    ).toBe(false);
  });

  it("preserves create keys and secrets across retries", () => {
    const idempotencyKey = crypto.randomUUID();
    const secret = generateSigningSecret();
    const create = { ...values, url: "https://example.com/hook" };
    const input = destinationCreateInput(create, idempotencyKey, secret);
    expect(destinationCreateInput(create, idempotencyKey, secret)).toEqual(input);
    expect(input.signingSecret).toBe(secret);
    expect(input.idempotencyKey).toBe(idempotencyKey);
    expect(input.url).toBe(create.url);
  });

  it("does not send custom headers or a signing secret to provider webhooks", () => {
    const input = destinationCreateInput(
      {
        ...values,
        channel: "slack",
        url: "https://hooks.slack.com/services/test",
        headers: [{ name: "X-Token", value: "secret" }],
      },
      crypto.randomUUID(),
      generateSigningSecret(),
    );
    expect(input.headers).toBeUndefined();
    expect(input.signingSecret).toBeUndefined();
  });

  it("generates a fresh signing secret within the API length limits", () => {
    const secret = generateSigningSecret();
    expect(secret).toMatch(/^whsec_[0-9a-f]{48}$/);
    expect(generateSigningSecret()).not.toBe(secret);
  });
});
