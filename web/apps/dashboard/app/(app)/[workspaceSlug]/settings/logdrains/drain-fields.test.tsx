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
