import { zodResolver } from "@hookform/resolvers/zod";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import React from "react";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, expect, it, vi } from "vitest";
import { EventTypesField } from "./drain-fields";
import { type DrainFormValues, createDrainSchema, emptyDrainForm } from "./drain-schema";

vi.stubGlobal("React", React);
vi.stubGlobal("PointerEvent", MouseEvent);
const sourceState = vi.hoisted(() => ({ loading: false, failed: false, combined: false }));
vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    deploy: {
      project: {
        list: {
          useQuery: () => ({
            data: [
              {
                id: "project",
                name: "Store",
                apps: [
                  { id: "app", name: "Backend" },
                  ...(sourceState.combined ? [{ id: "other-app", name: "Reports" }] : []),
                ],
              },
              {
                id: "other-project",
                name: "Analytics",
                apps: sourceState.combined ? [] : [{ id: "other-app", name: "Reports" }],
              },
            ],
            isLoading: sourceState.loading,
            error: sourceState.failed ? new Error("Unavailable") : null,
          }),
        },
      },
      environment: {
        listAll: {
          useQuery: () => ({
            data: [
              { id: "env", name: "Production", projectId: "project", appId: "app" },
              { id: "env_preview", name: "Preview", projectId: "project", appId: "app" },
              {
                id: "other-env",
                name: "Production",
                projectId: "other-project",
                appId: "other-app",
              },
            ],
            isLoading: false,
          }),
        },
      },
      environmentSettings: {
        getAvailableKeyspaces: {
          useQuery: () => ({
            data: {
              ks_primary: { id: "ks_primary", api: { name: "Production" } },
            },
            isLoading: false,
          }),
        },
      },
    },
  },
}));
afterEach(() => {
  cleanup();
  sourceState.loading = false;
  sourceState.failed = false;
  sourceState.combined = false;
});

function Form() {
  const form = useForm<DrainFormValues>({
    defaultValues: {
      ...emptyDrainForm,
      eventTypes: ["key.create"],
      eventTypesMode: "specific",
      outcomes: ["RATE_LIMITED"],
      statusClasses: [4],
    },
  });
  return (
    <FormProvider {...form}>
      <button type="button" onClick={() => form.setValue("stream", "audit_logs")}>
        Audit
      </button>
      <button type="button" onClick={() => form.setValue("stream", "key_verifications")}>
        Verifications
      </button>
      <EventTypesField />
      <button type="button" onClick={() => form.setValue("stream", "gateway_requests")}>
        Gateway
      </button>
    </FormProvider>
  );
}

it("keeps each stream filter bound to its own values when switching streams", () => {
  render(<Form />);
  expect(screen.getByLabelText("Search event types")).toBeTruthy();
  expect(screen.getByText("key.create")).toBeTruthy();
  expect(screen.queryByText("RATE_LIMITED")).toBeNull();
  const docs = screen.getByRole("link", { name: "View event types" });
  expect(docs.getAttribute("href")).toBe("https://www.unkey.com/docs/audit-log/types");
  expect(docs.getAttribute("target")).toBe("_blank");
  expect(docs.getAttribute("rel")).toBe("noopener noreferrer");

  fireEvent.click(screen.getByText("Verifications"));
  expect(screen.getByLabelText("Search outcomes")).toBeTruthy();
  expect(screen.getByText("RATE_LIMITED")).toBeTruthy();
  expect(screen.queryByText("key.create")).toBeNull();
  expect(screen.getByPlaceholderText("All keyspaces")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Remove" }));
  expect(screen.queryByText("RATE_LIMITED")).toBeNull();
  expect(screen.getByPlaceholderText("All outcomes")).toBeTruthy();

  fireEvent.click(screen.getByText("Audit"));
  expect(screen.getByLabelText("Search event types")).toBeTruthy();
  expect(screen.getByText("key.create")).toBeTruthy();
  expect(screen.queryByText("RATE_LIMITED")).toBeNull();

  fireEvent.click(screen.getByText("Verifications"));
  expect(screen.getByPlaceholderText("All outcomes")).toBeTruthy();
  expect(screen.queryByText("RATE_LIMITED")).toBeNull();
});

it("keeps gateway statuses on all until a mode asks for more", () => {
  render(<Form />);
  fireEvent.click(screen.getByText("Gateway"));
  expect(screen.getByRole("radio", { name: "All statuses" }).getAttribute("aria-checked")).toBe(
    "true",
  );
  expect(screen.queryByLabelText("Search HTTP statuses")).toBeNull();

  fireEvent.click(screen.getByRole("radio", { name: "Custom" }));
  expect(screen.getByLabelText("Search HTTP statuses")).toBeTruthy();
  expect(screen.getByText("4xx")).toBeTruthy();

  fireEvent.click(screen.getByRole("radio", { name: "Errors only" }));
  expect(screen.queryByLabelText("Search HTTP statuses")).toBeNull();
  fireEvent.click(screen.getByText("Verifications"));
  expect(screen.getByText("RATE_LIMITED")).toBeTruthy();
  fireEvent.click(screen.getByText("Gateway"));
  expect(screen.getByRole("radio", { name: "Errors only" }).getAttribute("aria-checked")).toBe(
    "true",
  );
});

it("ticks a whole project and reports a partial one", () => {
  const values: DrainFormValues[] = [];
  function TreeForm() {
    const form = useForm<DrainFormValues>({
      defaultValues: { ...emptyDrainForm, stream: "gateway_requests" },
    });
    values[0] = form.watch();
    return (
      <FormProvider {...form}>
        <EventTypesField />
      </FormProvider>
    );
  }
  render(<TreeForm />);
  expect(screen.getByText("All 3 environments")).toBeTruthy();

  fireEvent.click(screen.getByRole("checkbox", { name: /Analytics/ }));
  expect(screen.getByText("2 of 3 environments")).toBeTruthy();
  expect(values[0].sourceMode).toBe("some");
  expect(values[0].projectIds).toEqual(["project"]);
  expect(values[0].environmentIds).toEqual([]);

  fireEvent.click(screen.getByRole("checkbox", { name: /Preview/ }));
  expect(screen.getByRole("checkbox", { name: /Store/ }).getAttribute("aria-checked")).toBe(
    "mixed",
  );
  expect(values[0].environmentIds).toEqual(["env"]);
});

it("returns to every source when all of them are ticked again", () => {
  const values: DrainFormValues[] = [];
  function TreeForm() {
    const form = useForm<DrainFormValues>({
      defaultValues: {
        ...emptyDrainForm,
        stream: "gateway_requests",
        sourceMode: "some",
        appIds: ["app"],
      },
    });
    values[0] = form.watch();
    return (
      <FormProvider {...form}>
        <EventTypesField />
      </FormProvider>
    );
  }
  render(<TreeForm />);
  expect(screen.getByText("2 of 3 environments")).toBeTruthy();

  fireEvent.click(screen.getByRole("checkbox", { name: /Analytics/ }));
  expect(screen.getByText("All 3 environments")).toBeTruthy();
  expect(values[0].sourceMode).toBe("all");
  expect(values[0].appIds).toEqual([]);
});

it("shows nothing selected after clearing every source", () => {
  function TreeForm() {
    const form = useForm<DrainFormValues>({
      defaultValues: { ...emptyDrainForm, stream: "gateway_requests" },
    });
    return (
      <FormProvider {...form}>
        <EventTypesField />
      </FormProvider>
    );
  }
  render(<TreeForm />);
  fireEvent.click(screen.getByRole("button", { name: "Clear all" }));
  expect(screen.getByText("0 of 3 environments")).toBeTruthy();
  expect(screen.getByRole("checkbox", { name: /Store/ }).getAttribute("aria-checked")).toBe(
    "false",
  );
});

it.each(["loading", "failed"] as const)("does not change sources while queries are %s", (state) => {
  sourceState[state] = true;
  render(<Form />);
  fireEvent.click(screen.getByText("Gateway"));
  fireEvent.click(screen.getByRole("checkbox", { name: /Store/ }));
  fireEvent.click(screen.getByRole("button", { name: "Clear all" }));
  expect(screen.getByText("All 3 environments")).toBeTruthy();
});

it("keeps a searched project checkbox bound to every app in that project", () => {
  sourceState.combined = true;
  render(<Form />);
  fireEvent.click(screen.getByText("Gateway"));
  fireEvent.click(screen.getByRole("button", { name: "Clear all" }));
  fireEvent.click(screen.getByRole("checkbox", { name: /Backend/ }));
  fireEvent.change(screen.getByLabelText("Search sources"), { target: { value: "Backend" } });
  expect(screen.queryByRole("checkbox", { name: /Reports/ })).toBeNull();
  expect(screen.getByRole("checkbox", { name: /Store/ }).getAttribute("aria-checked")).toBe(
    "mixed",
  );
  fireEvent.click(screen.getByRole("checkbox", { name: /Store/ }));
  expect(screen.getByText("All 3 environments")).toBeTruthy();
});

function SubmitForm({
  onValid = () => {},
  eventTypes = [],
}: {
  onValid?: () => void;
  eventTypes?: string[];
}) {
  const form = useForm<DrainFormValues>({
    resolver: zodResolver(createDrainSchema),
    defaultValues: {
      ...emptyDrainForm,
      name: "Production audit logs",
      url: "https://example.com/ingest",
      eventTypesMode: "specific",
      eventTypes,
    },
    mode: "onChange",
  });
  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onValid)}>
        <EventTypesField />
        <button type="submit">Create</button>
      </form>
    </FormProvider>
  );
}

function clickCreate() {
  fireEvent.click(screen.getByRole("button", { name: "Create" }));
}

function chooseMode(name: string) {
  fireEvent.click(screen.getByRole("radio", { name }));
}

it("blocks submit when specific event types has no chosen type", async () => {
  const onValid = vi.fn();
  render(<SubmitForm onValid={onValid} />);

  clickCreate();

  await waitFor(() =>
    expect(screen.getByRole("alert").textContent).toBe("Choose at least one event type"),
  );
  expect(onValid).not.toHaveBeenCalled();
});

it("submits with no chosen type once all event types is picked", async () => {
  const onValid = vi.fn();
  render(<SubmitForm onValid={onValid} />);

  clickCreate();
  await waitFor(() => expect(screen.getByRole("alert")).toBeTruthy());

  chooseMode("All event types");
  expect(screen.queryByLabelText("Search event types")).toBeNull();
  expect(screen.queryByRole("alert")).toBeNull();

  clickCreate();
  await waitFor(() => expect(onValid).toHaveBeenCalledTimes(1));
});

it("asks again for a type when specific mode comes back empty", async () => {
  render(<SubmitForm />);

  clickCreate();
  await waitFor(() => expect(screen.getByRole("alert")).toBeTruthy());

  chooseMode("All event types");
  expect(screen.queryByRole("alert")).toBeNull();

  chooseMode("Specific event types");
  expect(screen.getByRole("alert").textContent).toBe("Choose at least one event type");
});

it("keeps the chosen event types when switching to all and back", async () => {
  render(<SubmitForm eventTypes={["key.create", "key.delete"]} />);
  expect(
    screen.getByText(`Sending 2 of ${unkeyAuditLogEvents.options.length} event types.`),
  ).toBeTruthy();

  chooseMode("All event types");
  expect(screen.queryByLabelText("Search event types")).toBeNull();

  chooseMode("Specific event types");
  expect(screen.getByText("key.create")).toBeTruthy();
  expect(screen.getByText("key.delete")).toBeTruthy();
  await waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
});
