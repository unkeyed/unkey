import { NextRequest } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  constructEvent: vi.fn(),
  contactsGet: vi.fn(),
  contactsCreate: vi.fn(),
  sendWelcomeEmail: vi.fn(),
  captureException: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  env: () => ({
    RESEND_API_KEY: "re_test",
    RESEND_AUDIENCE_ID: "aud_test",
    WORKOS_API_KEY: "sk_test",
    WORKOS_WEBHOOK_SECRET: "whsec_test",
  }),
}));

vi.mock("@sentry/nextjs", () => ({ captureException: mocks.captureException }));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  getWorkOS: () => ({ webhooks: { constructEvent: mocks.constructEvent } }),
}));

vi.mock("@unkey/resend", () => ({
  Resend: class {
    client = { contacts: { get: mocks.contactsGet, create: mocks.contactsCreate } };
    sendWelcomeEmail = mocks.sendWelcomeEmail;
  },
}));

import { POST } from "./route";

const NOT_FOUND = { name: "not_found", message: "contact not found" };

function verifiedUserCreated(overrides: Record<string, unknown> = {}) {
  return {
    event: "user.created",
    data: {
      id: "user_123",
      email: "jane@example.com",
      emailVerified: true,
      ...overrides,
    },
  };
}

function postWebhook() {
  return POST(
    new NextRequest("https://app.unkey.com/api/webhooks/workos", {
      method: "POST",
      headers: { "workos-signature": "signed" },
      body: "{}",
    }),
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.contactsGet.mockResolvedValue({ data: null, error: NOT_FOUND });
  mocks.contactsCreate.mockResolvedValue({ data: { id: "contact_123" } });
  mocks.sendWelcomeEmail.mockResolvedValue(undefined);
});

describe("POST /api/webhooks/workos", () => {
  it("returns 400 and does not report to Sentry when the signature fails", async () => {
    mocks.constructEvent.mockRejectedValue(new Error("invalid signature"));

    const response = await postWebhook();

    expect(response.status).toBe(400);
    expect(mocks.captureException).not.toHaveBeenCalled();
    expect(mocks.sendWelcomeEmail).not.toHaveBeenCalled();
  });

  it("returns 500 and reports to Sentry when a downstream provider call fails", async () => {
    mocks.constructEvent.mockResolvedValue(verifiedUserCreated());
    mocks.sendWelcomeEmail.mockRejectedValue(new Error("resend unavailable"));

    const response = await postWebhook();

    expect(response.status).toBe(500);
    expect(mocks.captureException).toHaveBeenCalledOnce();
    expect(mocks.contactsCreate).not.toHaveBeenCalled();
  });

  it("sends the welcome email before recording audience membership", async () => {
    mocks.constructEvent.mockResolvedValue(verifiedUserCreated());

    const response = await postWebhook();

    expect(response.status).toBe(200);
    expect(mocks.sendWelcomeEmail).toHaveBeenCalledWith({
      email: "jane@example.com",
      idempotencyKey: "workos-welcome:user_123",
    });
    expect(mocks.contactsCreate).toHaveBeenCalledWith({
      audienceId: "aud_test",
      email: "jane@example.com",
    });
    expect(mocks.sendWelcomeEmail.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.contactsCreate.mock.invocationCallOrder[0],
    );
  });

  it("does not welcome a user who is already in the audience", async () => {
    mocks.constructEvent.mockResolvedValue(verifiedUserCreated());
    mocks.contactsGet.mockResolvedValue({ data: { id: "contact_123" } });

    const response = await postWebhook();

    expect(response.status).toBe(200);
    expect(mocks.sendWelcomeEmail).not.toHaveBeenCalled();
    expect(mocks.contactsCreate).not.toHaveBeenCalled();
  });

  it("ignores an unverified user", async () => {
    mocks.constructEvent.mockResolvedValue(verifiedUserCreated({ emailVerified: false }));

    const response = await postWebhook();

    expect(response.status).toBe(200);
    expect(mocks.contactsGet).not.toHaveBeenCalled();
    expect(mocks.sendWelcomeEmail).not.toHaveBeenCalled();
  });
});
