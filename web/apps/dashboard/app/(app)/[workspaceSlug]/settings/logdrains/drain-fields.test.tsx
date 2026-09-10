import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, expect, it, vi } from "vitest";
import { EventTypesField } from "./drain-fields";
import { type DrainFormValues, emptyDrainForm } from "./drain-schema";

vi.stubGlobal("React", React);
vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    deploy: {
      project: {
        list: {
          useQuery: () => ({
            data: [
              { id: "project", name: "Store", apps: [{ id: "app", name: "Backend" }] },
              {
                id: "other-project",
                name: "Analytics",
                apps: [{ id: "other-app", name: "Reports" }],
              },
            ],
            isLoading: false,
          }),
        },
      },
      environment: {
        listAll: {
          useQuery: () => ({
            data: [{ id: "env", name: "Production", projectId: "project", appId: "app" }],
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
afterEach(cleanup);

function Form() {
  const form = useForm<DrainFormValues>({
    defaultValues: {
      ...emptyDrainForm,
      eventTypes: ["key.create"],
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

it("keeps gateway status selection separate and retains clearing after switching", () => {
  render(<Form />);
  fireEvent.click(screen.getByText("Gateway"));
  expect(screen.getByLabelText("Search HTTP statuses")).toBeTruthy();
  expect(screen.getByText("4xx")).toBeTruthy();
  expect(screen.queryByText("key.create")).toBeNull();
  fireEvent.click(screen.getByText("Verifications"));
  expect(screen.getByText("RATE_LIMITED")).toBeTruthy();
  fireEvent.click(screen.getByText("Gateway"));
  expect(screen.getByText("4xx")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Remove" }));
  expect(screen.getByPlaceholderText("All HTTP statuses")).toBeTruthy();
  fireEvent.click(screen.getByText("Audit"));
  expect(screen.getByText("key.create")).toBeTruthy();
  fireEvent.click(screen.getByText("Gateway"));
  expect(screen.getByPlaceholderText("All HTTP statuses")).toBeTruthy();
});

it("retains resource selections across streams and clears each independently", () => {
  function SelectedForm() {
    const form = useForm<DrainFormValues>({
      defaultValues: {
        ...emptyDrainForm,
        stream: "gateway_requests",
        projectIds: ["project"],
        appIds: ["app"],
        environmentIds: ["env"],
        statusClasses: [5],
      },
    });
    return (
      <FormProvider {...form}>
        <button type="button" onClick={() => form.setValue("stream", "audit_logs")}>
          Audit
        </button>
        <button type="button" onClick={() => form.setValue("stream", "gateway_requests")}>
          Gateway
        </button>
        <EventTypesField />
      </FormProvider>
    );
  }
  render(<SelectedForm />);
  expect(screen.getByText("Store (project)")).toBeTruthy();
  expect(screen.getByText("Store / Backend (app)")).toBeTruthy();
  expect(screen.getByText("Store / Backend / Production (env)")).toBeTruthy();
  fireEvent.click(screen.getByText("Audit"));
  expect(screen.queryByLabelText("Search projects")).toBeNull();
  fireEvent.click(screen.getByText("Gateway"));
  expect(screen.getByText("Store (project)")).toBeTruthy();
  for (const label of ["Search projects", "Search apps", "Search environments"]) {
    const fieldset = screen.getByLabelText(label).closest("fieldset");
    const button = fieldset?.querySelector("button[aria-label='Remove']");
    if (!button) {
      throw new Error(`Missing remove button for ${label}`);
    }
    fireEvent.click(button);
  }
  expect(screen.getByText("5xx")).toBeTruthy();
  fireEvent.click(screen.getByText("Audit"));
  fireEvent.click(screen.getByText("Gateway"));
  expect(screen.getByPlaceholderText("All projects")).toBeTruthy();
  expect(screen.getByPlaceholderText("All apps")).toBeTruthy();
  expect(screen.getByPlaceholderText("All environments")).toBeTruthy();
});

it("scopes apps to selected projects and removes selected apps outside that scope", async () => {
  function SelectedForm() {
    const form = useForm<DrainFormValues>({
      defaultValues: { ...emptyDrainForm, stream: "gateway_requests", appIds: ["other-app"] },
    });
    return (
      <FormProvider {...form}>
        <EventTypesField />
      </FormProvider>
    );
  }
  render(<SelectedForm />);
  expect(screen.getByText("Analytics / Reports (other-app)")).toBeTruthy();
  fireEvent.keyDown(screen.getByLabelText("Search projects"), { key: "ArrowDown" });
  fireEvent.click(await screen.findByRole("option", { name: "Store (project)" }));
  expect(screen.queryByText("Analytics / Reports (other-app)")).toBeNull();
  fireEvent.keyDown(screen.getByLabelText("Search apps"), { key: "ArrowDown" });
  expect(await screen.findByRole("option", { name: "Store / Backend (app)" })).toBeTruthy();
  expect(screen.queryByRole("option", { name: "Analytics / Reports (other-app)" })).toBeNull();
  fireEvent.keyDown(screen.getByLabelText("Search apps"), { key: "Escape" });
  const remove = screen
    .getByLabelText("Search projects")
    .closest("fieldset")
    ?.querySelector("button[aria-label='Remove']");
  if (!remove) {
    throw new Error("Missing project remove button");
  }
  fireEvent.click(remove);
  fireEvent.keyDown(screen.getByLabelText("Search apps"), { key: "ArrowDown" });
  expect(
    await screen.findByRole("option", { name: "Analytics / Reports (other-app)" }),
  ).toBeTruthy();
  expect(screen.getByRole("option", { name: "Store / Backend (app)" })).toBeTruthy();
});
