import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ConflictErrorResponse } from "@unkey/api/models/errors";
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from "react";
import { forwardRef } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CreatePortalDialog } from "./create-portal-dialog";

const SLUG_CONFLICT_DETAIL = "That slug is already in use. Choose a different slug.";
const MAPPING_CONFLICT_DETAIL = "That app or keyspace already has a portal.";

/**
 * A real `ConflictErrorResponse`: `portalConflict` gates on the response class
 * before it looks at the detail, so a bare Error would not be classified.
 */
function conflict(detail: string): ConflictErrorResponse {
  return new ConflictErrorResponse(
    {
      meta: { requestId: "req_test" },
      error: {
        title: "Conflict",
        detail,
        status: 409,
        type: "https://unkey.com/docs/errors/data/portal/duplicate",
      },
    },
    {
      request: new Request("https://api.unkey.com/v2/portal"),
      response: new Response(null, { status: 409 }),
      body: "",
    },
  );
}

const mocks = vi.hoisted(() => ({
  mutateAsync: vi.fn(),
  getPortalByMapping: vi.fn(),
  invalidateQueries: vi.fn(),
  setQueryData: vi.fn(),
}));

vi.mock("@/lib/portal/use-portal", () => ({
  useCreatePortal: () => ({ mutateAsync: mocks.mutateAsync }),
  portalQueryKey: (keyAuthId: string) => ["portal", keyAuthId],
}));

vi.mock("@/lib/portal/client", () => ({
  getPortalByMapping: mocks.getPortalByMapping,
  keyspaceMapping: (id: string) => ({ type: "keyspace", id }),
}));

vi.mock("@/lib/unkey-client", async () => {
  const { ConflictErrorResponse: Conflict } = await import("@unkey/api/models/errors");
  return {
    getErrorMessage: (error: unknown) =>
      error instanceof Conflict ? error.error.detail : "Something went wrong",
  };
});

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    invalidateQueries: mocks.invalidateQueries,
    setQueryData: mocks.setQueryData,
  }),
}));

vi.mock("@unkey/ui", () => ({
  DialogContainer: ({ children, footer }: { children?: ReactNode; footer?: ReactNode }) => (
    <div>
      {children}
      {footer}
    </div>
  ),
  Button: ({
    children,
    loading: _loading,
    loadingLabel: _loadingLabel,
    variant: _variant,
    size: _size,
    ...props
  }: ButtonHTMLAttributes<HTMLButtonElement> & {
    loading?: boolean;
    loadingLabel?: string;
    variant?: string;
    size?: string;
  }) => <button {...props}>{children}</button>,
  FormInput: forwardRef<
    HTMLInputElement,
    InputHTMLAttributes<HTMLInputElement> & { label?: string; error?: string; description?: string }
  >(({ label, error, description: _description, ...props }, ref) => (
    <span>
      <label htmlFor="slug">{label}</label>
      <input id="slug" ref={ref} {...props} />
      {error ? <span data-testid="slug-error">{error}</span> : null}
    </span>
  )),
}));

const onOpenChange = vi.fn();

function renderDialog(resourceName = "Payments API") {
  return render(
    <CreatePortalDialog
      keyAuthId="ks_123"
      resourceName={resourceName}
      isOpen
      onOpenChange={onOpenChange}
    />,
  );
}

function slugInput(): HTMLInputElement {
  const input = screen.getByLabelText("Portal slug");
  if (!(input instanceof HTMLInputElement)) {
    throw new Error("slug field is not an input");
  }
  return input;
}

function submitForm() {
  fireEvent.click(screen.getByRole("button", { name: "Enable portal" }));
}

describe("CreatePortalDialog", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    mocks.mutateAsync.mockResolvedValue({ portalId: "portal_123" });
    mocks.getPortalByMapping.mockRejectedValue(new Error("not found"));
  });

  it("prefills the slug from the API name", () => {
    renderDialog();
    expect(slugInput().value).toBe("payments-api");
  });

  it("creates the portal and closes on success", async () => {
    renderDialog();

    submitForm();

    await waitFor(() => expect(mocks.mutateAsync).toHaveBeenCalled());
    expect(mocks.mutateAsync).toHaveBeenCalledWith({ slug: "payments-api", enabled: true });
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    // The re-read is `useCreatePortal`'s invalidate, not a second one here.
    expect(mocks.invalidateQueries).not.toHaveBeenCalled();
  });

  it("blocks an invalid slug without sending a request", async () => {
    renderDialog();

    fireEvent.change(slugInput(), { target: { value: "my--portal" } });
    submitForm();

    expect(await screen.findByTestId("slug-error")).toBeDefined();
    expect(mocks.mutateAsync).not.toHaveBeenCalled();
  });

  it("puts a slug conflict on the field after re-reading by mapping", async () => {
    mocks.mutateAsync.mockRejectedValue(conflict(SLUG_CONFLICT_DETAIL));
    renderDialog();

    submitForm();

    expect((await screen.findByTestId("slug-error")).textContent).toBe(SLUG_CONFLICT_DETAIL);
    // A 409 can also mean the create landed and the ack was lost, so the field
    // error is only reported once a re-read has failed to find a portal.
    expect(mocks.getPortalByMapping).toHaveBeenCalledWith({ type: "keyspace", id: "ks_123" });
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("treats a slug conflict as success when the re-read finds a portal", async () => {
    mocks.mutateAsync.mockRejectedValue(conflict(SLUG_CONFLICT_DETAIL));
    mocks.getPortalByMapping.mockResolvedValue({ id: "portal_123", slug: "payments-api" });
    renderDialog();

    submitForm();

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    // The re-read already carries the row, so it seeds the cache rather than
    // triggering a third fetch of the same resource.
    expect(mocks.setQueryData).toHaveBeenCalledWith(["portal", "ks_123"], {
      found: true,
      portal: { id: "portal_123", slug: "payments-api" },
    });
    expect(screen.queryByTestId("slug-error")).toBeNull();
  });

  it("reports a mapping conflict at dialog level, not on the slug field", async () => {
    mocks.mutateAsync.mockRejectedValue(conflict(MAPPING_CONFLICT_DETAIL));
    renderDialog();

    submitForm();

    expect(await screen.findByText(/already has a customer portal/i)).toBeDefined();
    // No slug can win a mapping conflict, so the field carries no error and the
    // owning portal is never re-read for.
    expect(screen.queryByTestId("slug-error")).toBeNull();
    expect(mocks.getPortalByMapping).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("shows a non-conflict failure inside the dialog", async () => {
    mocks.mutateAsync.mockRejectedValue(new Error("network down"));
    renderDialog();

    submitForm();

    expect(await screen.findByText("Something went wrong")).toBeDefined();
    expect(onOpenChange).not.toHaveBeenCalled();
  });
});
