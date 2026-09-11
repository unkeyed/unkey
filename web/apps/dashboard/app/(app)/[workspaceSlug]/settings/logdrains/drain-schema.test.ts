import { describe, expect, it } from "vitest";
import {
  type DrainFormValues,
  createDrainSchema,
  drainToFormValues,
  editDrainSchema,
  emptyDrainForm,
  submittedEventTypes,
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
  it.each([createDrainSchema, editDrainSchema])(
    "rejects an empty runtime selection without using gateway sources",
    (schema) => {
      expect(
        messagesFor(schema, {
          ...httpDrain,
          stream: "runtime_logs",
          runtimeSourceMode: "some",
          sourceMode: "some",
          projectIds: ["gateway-project"],
          statusMode: "custom",
        }),
      ).toEqual(["Choose at least one source"]);
    },
  );

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

  it("rejects specific event types with an empty list", () => {
    expect(messagesFor(createDrainSchema, { ...httpDrain, eventTypesMode: "specific" })).toContain(
      "Choose at least one event type",
    );
  });

  it("accepts specific event types with a chosen type", () => {
    expect(
      messagesFor(createDrainSchema, {
        ...httpDrain,
        eventTypesMode: "specific",
        eventTypes: ["key.create"],
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

  it("loads saved 5xx and 4xx filters as custom without changing them", () => {
    const values = drainToFormValues({
      id: "drain",
      name: "Gateway",
      status: "running",
      kind: "http",
      stream: "gateway_requests",
      config: { url: "https://example.com/ingest", format: "ndjson", headers: [] },
      eventTypes: [],
      outcomes: [],
      keySpaceIds: [],
      statusClasses: [5, 4],
      severities: [],
      projectIds: [],
      appIds: [],
      environmentIds: [],
    });
    expect(values.statusMode).toBe("custom");
    expect(submittedStatusClasses(values)).toEqual([5, 4]);
  });

  it("sends the chosen classes in custom mode", () => {
    expect(
      submittedStatusClasses({ ...emptyDrainForm, statusMode: "custom", statusClasses: [3] }),
    ).toEqual([3]);
  });
});

describe("submittedSources", () => {
  it("initializes stored runtime restrictions in the runtime mode", () => {
    const values = drainToFormValues({
      id: "drain",
      name: "Runtime",
      status: "running",
      kind: "http",
      stream: "runtime_logs",
      config: { url: "https://example.com/ingest", format: "ndjson", headers: ["Authorization"] },
      eventTypes: [],
      outcomes: [],
      keySpaceIds: [],
      statusClasses: [],
      severities: ["warn"],
      projectIds: ["deleted-project"],
      appIds: [],
      environmentIds: [],
    });
    expect(values.runtimeSourceMode).toBe("some");
    expect(values.sourceMode).toBe("all");
    expect(submittedSources(values)).toEqual({
      projectIds: ["deleted-project"],
      appIds: [],
      environmentIds: [],
    });
  });

  it("uses runtime sources even when the gateway mode is all", () => {
    expect(
      submittedSources({
        ...emptyDrainForm,
        stream: "runtime_logs",
        runtimeSourceMode: "some",
        runtimeProjectIds: ["runtime-project"],
        runtimeEnvironmentIds: ["deleted-environment"],
        sourceMode: "all",
        projectIds: ["gateway-project"],
      }),
    ).toEqual({
      projectIds: ["runtime-project"],
      appIds: [],
      environmentIds: ["deleted-environment"],
    });
  });

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

describe("event type validation", () => {
  it("ignores the event type mode on the key verifications stream", () => {
    expect(
      messagesFor(createDrainSchema, {
        ...httpDrain,
        stream: "key_verifications",
        eventTypesMode: "specific",
      }),
    ).toEqual([]);
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

describe("submittedEventTypes", () => {
  it("sends no event types in all mode, whatever the kept selection is", () => {
    expect(
      submittedEventTypes({
        ...emptyDrainForm,
        eventTypesMode: "all",
        eventTypes: ["key.create", "key.delete"],
      }),
    ).toEqual([]);
  });

  it("sends the chosen event types in specific mode", () => {
    expect(
      submittedEventTypes({
        ...emptyDrainForm,
        eventTypesMode: "specific",
        eventTypes: ["key.create"],
      }),
    ).toEqual(["key.create"]);
  });
});
