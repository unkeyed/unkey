import { describe, expect, it } from "vitest";
import {
  type DrainFormValues,
  createDrainSchema,
  editDrainSchema,
  emptyDrainForm,
  submittedSources,
  submittedStatusClasses,
} from "./drain-schema";

function messagesFor(schema: typeof createDrainSchema, values: Partial<DrainFormValues>): string[] {
  const result = schema.safeParse({ ...emptyDrainForm, ...values });
  return result.success ? [] : result.error.issues.map((issue) => issue.message);
}

const httpDrain = {
  kind: "http",
  name: "Production audit logs",
  url: "https://example.com/ingest",
} satisfies Partial<DrainFormValues>;

describe("createDrainSchema", () => {
  it("does not accept a historical delivery start offset", () => {
    const result = createDrainSchema.safeParse({
      ...emptyDrainForm,
      ...httpDrain,
      startFrom: "beginning",
    });

    expect(result.success && "startFrom" in result.data).toBe(false);
  });

  it("accepts an HTTP drain with a single blank header row", () => {
    expect(messagesFor(createDrainSchema, httpDrain)).toEqual([]);
  });

  it.each([
    ["not a url", "Enter a valid URL"],
    ["http://example.com/ingest", "URL must use HTTPS"],
    ["https://user:pass@example.com/ingest", "URL must not contain credentials"],
  ])("rejects %s", (url, message) => {
    expect(messagesFor(createDrainSchema, { ...httpDrain, url })).toContain(message);
  });

  it.each([
    [[{ name: "", value: "Bearer token", stored: false }], "Enter a header name"],
    [[{ name: "Authorization", value: "", stored: false }], "Enter a header value"],
    [[{ name: "Invalid Header", value: "value", stored: false }], "Enter a valid header name"],
    [
      [{ name: "Authorization", value: "Bearer token\r\nX-Injected: value", stored: false }],
      "Enter a valid header value",
    ],
    [
      [
        { name: "Authorization", value: "one", stored: false },
        { name: "authorization", value: "two", stored: false },
      ],
      "Header name is duplicated",
    ],
  ])("rejects invalid header rows", (headers, message) => {
    expect(messagesFor(createDrainSchema, { ...httpDrain, headers })).toContain(message);
  });

  it("requires a name", () => {
    expect(messagesFor(createDrainSchema, { ...httpDrain, name: "  " })).toContain("Enter a name");
  });

  it("requires an Axiom dataset and token", () => {
    const messages = messagesFor(createDrainSchema, { kind: "axiom", name: "Axiom" });
    expect(messages).toContain("Enter a dataset");
    expect(messages).toContain("Enter a token");
  });

  it("ignores the unused destination's fields", () => {
    expect(messagesFor(createDrainSchema, { ...httpDrain, dataset: "", token: "" })).toEqual([]);
  });

  it("rejects a gateway drain whose sources are all unticked", () => {
    expect(
      messagesFor(createDrainSchema, {
        ...httpDrain,
        stream: "gateway_requests",
        sourceMode: "some",
      }),
    ).toContain("Choose at least one source");
  });

  it("accepts a gateway drain with a chosen source", () => {
    expect(
      messagesFor(createDrainSchema, {
        ...httpDrain,
        stream: "gateway_requests",
        sourceMode: "some",
        projectIds: ["project"],
      }),
    ).toEqual([]);
  });

  it("rejects custom statuses with an empty list", () => {
    expect(
      messagesFor(createDrainSchema, {
        ...httpDrain,
        stream: "gateway_requests",
        statusMode: "custom",
      }),
    ).toContain("Choose at least one status class");
  });

  it("ignores the gateway modes on other streams", () => {
    expect(
      messagesFor(createDrainSchema, {
        ...httpDrain,
        stream: "audit_logs",
        sourceMode: "some",
        statusMode: "custom",
      }),
    ).toEqual([]);
  });
});

describe("submittedStatusClasses", () => {
  it("sends nothing in all mode", () => {
    expect(
      submittedStatusClasses({ ...emptyDrainForm, statusMode: "all", statusClasses: [2] }),
    ).toEqual([]);
  });

  it("sends 4xx and 5xx in errors mode", () => {
    expect(submittedStatusClasses({ ...emptyDrainForm, statusMode: "errors" })).toEqual([4, 5]);
  });

  it("sends the chosen classes in custom mode", () => {
    expect(
      submittedStatusClasses({ ...emptyDrainForm, statusMode: "custom", statusClasses: [3] }),
    ).toEqual([3]);
  });
});

describe("submittedSources", () => {
  it("sends no filter in all mode, whatever the kept selection is", () => {
    expect(
      submittedSources({ ...emptyDrainForm, sourceMode: "all", projectIds: ["project"] }),
    ).toEqual({ projectIds: [], appIds: [], environmentIds: [] });
  });

  it("sends the encoded selection in some mode", () => {
    expect(submittedSources({ ...emptyDrainForm, sourceMode: "some", appIds: ["app"] })).toEqual({
      projectIds: [],
      appIds: ["app"],
      environmentIds: [],
    });
  });
});

describe("editDrainSchema", () => {
  it("keeps a stored header whose value is left blank", () => {
    const headers = [{ name: "Authorization", value: "", stored: true }];
    expect(messagesFor(editDrainSchema, { ...httpDrain, headers })).toEqual([]);
  });

  it("keeps the stored Axiom token when the field is left blank", () => {
    expect(
      messagesFor(editDrainSchema, { kind: "axiom", name: "Axiom", dataset: "audit-logs" }),
    ).toEqual([]);
  });
});
