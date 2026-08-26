import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactElement, ReactNode } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PolicyList } from "./policy-list";
import { type RootKeyFormValues, rootKeyDefaultValues } from "./schema";

// @unkey/ui declares react as a dependency rather than a peer, so under vitest
// its primitives run against a second React copy and every hook call throws.
// Only the hook-using primitives are stubbed; icons render for real.
vi.mock("@unkey/ui", () => ({
  Button: ({ children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
  Checkbox: ({
    id,
    checked,
    onCheckedChange,
  }: {
    id?: string;
    checked?: boolean;
    onCheckedChange: (checked: boolean) => void;
  }) => (
    <input
      id={id}
      type="checkbox"
      checked={checked}
      onChange={(event) => onCheckedChange(event.currentTarget.checked)}
    />
  ),
  Input: (props: InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
  InfoTooltip: ({ children }: { children: ReactNode }) => children,
  Item: ({ render }: { render: ReactElement }) => render,
  ItemContent: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  ItemDescription: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  ItemMedia: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  ItemTitle: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  Select: ({ value }: { value: string }) => <span aria-label="Resource type">{value}</span>,
  SelectContent: () => null,
  SelectItem: () => null,
  SelectTrigger: () => null,
  SelectValue: () => null,
}));

vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    deploy: {
      project: { list: { useQuery: () => ({ data: [], isLoading: false }) } },
      environment: { listAll: { useQuery: () => ({ data: [], isLoading: false }) } },
      environmentSettings: {
        getAvailableKeyspaces: { useQuery: () => ({ data: {}, isLoading: false }) },
      },
    },
    ratelimit: { namespace: { list: { useQuery: () => ({ data: [], isLoading: false }) } } },
  },
}));

afterEach(cleanup);

function ScopeProbe() {
  const policies = useWatch<RootKeyFormValues, "policies">({ name: "policies" });
  return <output>{policies.map((policy) => policy.scope).join(",")}</output>;
}

function Harness() {
  const form = useForm<RootKeyFormValues>({ defaultValues: rootKeyDefaultValues });
  return (
    <FormProvider {...form}>
      <PolicyList />
      <ScopeProbe />
    </FormProvider>
  );
}

const pick = (title: string) => fireEvent.click(screen.getByText(title));

const isEditorOpen = () => screen.queryByText("Edit policy") !== null;

const isGalleryOpen = () => screen.queryByText("Start new") !== null;

describe("PolicyList", () => {
  it("opens on the gallery and offers no add button yet", () => {
    render(<Harness />);

    expect(isGalleryOpen()).toBe(true);
    expect(screen.queryByText("Add policy")).toBeNull();
  });

  it("lands a template collapsed and writes it into the form", () => {
    render(<Harness />);
    pick("All read permissions");

    expect(isGalleryOpen()).toBe(false);
    expect(isEditorOpen()).toBe(false);
    expect(screen.getByText("All resources")).toBeDefined();
    expect(screen.getByRole("status").textContent).toBe("workspace");
  });

  it("lands a blank policy open", () => {
    render(<Harness />);
    pick("Start new");

    expect(isEditorOpen()).toBe(true);
  });

  it("collapses the open cards and reopens the gallery on add", () => {
    render(<Harness />);
    pick("Start new");
    fireEvent.click(screen.getByText("Add policy"));

    expect(isEditorOpen()).toBe(false);
    expect(isGalleryOpen()).toBe(true);
  });

  it("appends rather than replaces", () => {
    render(<Harness />);
    pick("All read permissions");
    fireEvent.click(screen.getByText("Add policy"));
    pick("Verify keys");

    expect(screen.getByRole("status").textContent).toBe("workspace,keyspaces");
  });

  it("dismisses the gallery once a policy exists", () => {
    render(<Harness />);
    pick("All read permissions");
    fireEvent.click(screen.getByText("Add policy"));
    fireEvent.click(screen.getByLabelText("Close templates"));

    expect(isGalleryOpen()).toBe(false);
  });

  it("returns to the gallery when the last policy is trashed", () => {
    render(<Harness />);
    pick("All read permissions");
    fireEvent.click(screen.getByLabelText("Delete policy"));

    expect(isGalleryOpen()).toBe(true);
    expect(screen.getByRole("status").textContent).toBe("");
  });
});
