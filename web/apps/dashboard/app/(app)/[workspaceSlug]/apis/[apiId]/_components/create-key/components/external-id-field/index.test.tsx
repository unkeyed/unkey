import { fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes, ChangeEvent, ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ExternalIdField } from ".";

const mocks = vi.hoisted(() => ({
  identitiesQuery: {
    identities: [],
    isFetching: false,
    isFetchingNextPage: false,
    hasNextPage: false,
    isLoading: false,
    isError: false,
    error: new Error(""),
    fetchNextPage: vi.fn(() => Promise.resolve()),
    refetch: vi.fn(() => Promise.resolve()),
  },
  createMutation: {
    isLoading: false,
    isError: false,
    error: new Error(""),
    reset: vi.fn(),
    mutateAsync: vi.fn(),
  },
}));

vi.mock("@/lib/identities-query", () => ({
  useIdentities: () => mocks.identitiesQuery,
  useCreateIdentityMutation: () => mocks.createMutation,
}));

vi.mock("@/components/ui/form-combobox", () => ({
  FormCombobox: ({
    disabled,
    emptyMessage,
    error,
    onChange,
  }: {
    disabled?: boolean;
    emptyMessage?: ReactNode;
    error?: string;
    onChange: (event: ChangeEvent<HTMLInputElement>) => void;
  }) => (
    <div>
      <input aria-label="External ID" disabled={disabled} onChange={onChange} />
      {error ? <div role="alert">{error}</div> : null}
      {emptyMessage}
    </div>
  ),
}));

vi.mock("@unkey/ui", () => ({
  Button: ({
    children,
    loading: _loading,
    ...props
  }: ButtonHTMLAttributes<HTMLButtonElement> & { loading?: boolean }) => (
    <button {...props}>{children}</button>
  ),
}));

describe("ExternalIdField feedback", () => {
  beforeEach(() => {
    mocks.identitiesQuery.isError = false;
    mocks.identitiesQuery.error = new Error("");
    mocks.identitiesQuery.refetch.mockClear();
    mocks.createMutation.isLoading = false;
    mocks.createMutation.isError = false;
    mocks.createMutation.error = new Error("");
    mocks.createMutation.reset.mockClear();
  });

  it("locks the selector while creating an identity", () => {
    mocks.createMutation.isLoading = true;

    render(<ExternalIdField value={null} onChange={vi.fn()} />);

    expect(screen.getByRole<HTMLInputElement>("textbox", { name: "External ID" }).disabled).toBe(
      true,
    );
  });

  it("shows identity query errors with an in-context retry action", () => {
    mocks.identitiesQuery.isError = true;
    mocks.identitiesQuery.error = new Error("Identity service unavailable");

    render(<ExternalIdField value={null} onChange={vi.fn()} />);

    expect(screen.getByRole("alert").textContent).toContain("We couldn't load identities.");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(mocks.identitiesQuery.refetch).toHaveBeenCalledOnce();
  });
});
