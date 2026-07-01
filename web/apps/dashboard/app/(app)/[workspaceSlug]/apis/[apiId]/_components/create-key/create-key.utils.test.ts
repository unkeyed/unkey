import { describe, expect, it } from "vitest";
import { type FormValues, formSchema } from "./create-key.schema";
import { formValuesToCreateKeyRequest, getDefaultValues } from "./create-key.utils";

function formValues(overrides: Partial<FormValues> = {}): FormValues {
  return formSchema.parse({
    ...getDefaultValues(),
    ...overrides,
  });
}

describe("formValuesToCreateKeyRequest", () => {
  it("maps minimal form values to the SDK request", () => {
    expect(formValuesToCreateKeyRequest(formValues(), "api_123")).toEqual({
      apiId: "api_123",
      byteLength: 16,
      enabled: true,
      recoverable: false,
    });
  });

  it("preserves configured key defaults and recoverable storage", () => {
    const values = formValues({
      bytes: 32,
      prefix: "prod",
      name: "Production",
      externalId: "user_123",
    });

    expect(formValuesToCreateKeyRequest(values, "api_123", { recoverable: true })).toEqual({
      apiId: "api_123",
      byteLength: 32,
      enabled: true,
      recoverable: true,
      prefix: "prod",
      name: "Production",
      externalId: "user_123",
    });
  });

  it("maps enabled metadata and expiration", () => {
    const expires = new Date(Date.now() + 10 * 60_000);
    const values = formValues({
      metadata: {
        enabled: true,
        data: JSON.stringify({ plan: "pro", seats: 5 }),
      },
      expiration: {
        enabled: true,
        data: expires,
      },
    });

    expect(formValuesToCreateKeyRequest(values, "api_123")).toMatchObject({
      apiId: "api_123",
      meta: { plan: "pro", seats: 5 },
      expires: expires.getTime(),
    });
  });

  it("maps credits without refill", () => {
    const values = formValues({
      limit: {
        enabled: true,
        data: {
          remaining: 1000,
          refill: {
            interval: "none",
          },
        },
      },
    });

    expect(formValuesToCreateKeyRequest(values, "api_123")).toMatchObject({
      credits: {
        remaining: 1000,
      },
    });
  });

  it("maps daily and monthly credit refills", () => {
    const daily = formValues({
      limit: {
        enabled: true,
        data: {
          remaining: 1000,
          refill: {
            interval: "daily",
            amount: 100,
            refillDay: undefined,
          },
        },
      },
    });
    const monthly = formValues({
      limit: {
        enabled: true,
        data: {
          remaining: 1000,
          refill: {
            interval: "monthly",
            amount: 500,
            refillDay: 15,
          },
        },
      },
    });

    expect(formValuesToCreateKeyRequest(daily, "api_123")).toMatchObject({
      credits: {
        remaining: 1000,
        refill: {
          interval: "daily",
          amount: 100,
        },
      },
    });
    expect(formValuesToCreateKeyRequest(monthly, "api_123")).toMatchObject({
      credits: {
        remaining: 1000,
        refill: {
          interval: "monthly",
          amount: 500,
          refillDay: 15,
        },
      },
    });
  });

  it("maps ratelimits to SDK field names", () => {
    const values = formValues({
      ratelimit: {
        enabled: true,
        data: [
          {
            name: "requests",
            limit: 10,
            refillInterval: 60_000,
            autoApply: true,
          },
        ],
      },
    });

    expect(formValuesToCreateKeyRequest(values, "api_123")).toMatchObject({
      ratelimits: [
        {
          name: "requests",
          limit: 10,
          duration: 60_000,
          autoApply: true,
        },
      ],
    });
  });
});
