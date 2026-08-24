import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ConflictErrorResponse, NotFoundErrorResponse } from "@unkey/api/models/errors";
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from "react";
import { forwardRef } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CreatePortalDialog } from "./create-portal-dialog";

const SLUG_CONFLICT_DETAIL = "That slug is already in use. Choose a different slug.";
const MAPPING_CONFLICT_DETAIL = "That app or keyspace already has a portal.";
/** The unique-index arm, which names no index and so could be either. */
const AMBIGUOUS_CONFLICT_DETAIL = "A portal already exists for that slug, app, or keyspace.";

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

/** Only a 404 proves the keyspace has no portal; anything else is undetermined. */
function notFound(): NotFoundErrorResponse {
  return new NotFoundErrorResponse(
    {
      meta: { requestId: "req_test" },
      error: {
        title: "Not Found",
        detail: "No portal found.",
        status: 404,
        type: "https://unkey.com/docs/errors/data/portal/not_found",
      },
    },
    {
      request: new Request("https://api.unkey.com/v2/portal"),
      response: new Response(null, { status: 404 }),
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
  >(({ label, error, description: _description, ...props }, ref) => {
    // The form has more than one field, so the id has to be per-label or every
    // `getByLabelText` would resolve to the first input.
    const id = (label ?? "field").toLowerCase().replace(/\s+/g, "-");
    return (
      <span>
        <label htmlFor={id}>{label}</label>
        <input id={id} ref={ref} {...props} />
        {error ? <span data-testid={`${id}-error`}>{error}</span> : null}
      </span>
    );
  }),
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

function inputFor(label: string): HTMLInputElement {
  const input = screen.getByLabelText(label);
  if (!(input instanceof HTMLInputElement)) {
    throw new Error(`${label} field is not an input`);
  }
  return input;
}

function slugInput(): HTMLInputElement {
  return inputFor("Portal slug");
}

function displayNameInput(): HTMLInputElement {
  return inputFor("Display name");
}

function submitForm() {
  fireEvent.click(screen.getByRole("button", { name: "Enable portal" }));
}

describe("CreatePortalDialog", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    mocks.mutateAsync.mockResolvedValue({ portalId: "portal_123" });
    mocks.getPortalByMapping.mockRejectedValue(notFound());
  });

  it("prefills both name fields from the API name", () => {
    renderDialog();
    expect(displayNameInput().value).toBe("Payments API");
    expect(slugInput().value).toBe("payments-api");
  });

  it("re-slugifies the slug while the operator edits the display name", async () => {
    renderDialog();

    fireEvent.change(displayNameInput(), { target: { value: "Acme Inc" } });

    await waitFor(() => expect(slugInput().value).toBe("acme-inc"));
  });

  it("stops following the display name once the slug is edited by hand", async () => {
    renderDialog();

    fireEvent.change(slugInput(), { target: { value: "chosen-slug" } });
    fireEvent.change(displayNameInput(), { target: { value: "Acme Inc" } });

    await waitFor(() => expect(displayNameInput().value).toBe("Acme Inc"));
    expect(slugInput().value).toBe("chosen-slug");
  });

  it("creates the portal and closes on success", async () => {
    renderDialog();

    submitForm();

    await waitFor(() => expect(mocks.mutateAsync).toHaveBeenCalled());
    expect(mocks.mutateAsync).toHaveBeenCalledWith({
      slug: "payments-api",
      displayName: "Payments API",
      enabled: true,
    });
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    // The re-read is `useCreatePortal`'s invalidate, not a second one here.
    expect(mocks.invalidateQueries).not.toHaveBeenCalled();
  });

  it("blocks an invalid slug without sending a request", async () => {
    renderDialog();

    fireEvent.change(slugInput(), { target: { value: "my--portal" } });
    submitForm();

    expect(await screen.findByTestId("portal-slug-error")).toBeDefined();
    expect(mocks.mutateAsync).not.toHaveBeenCalled();
  });

  it("puts a slug conflict on the field after re-reading by mapping", async () => {
    mocks.mutateAsync.mockRejectedValue(conflict(SLUG_CONFLICT_DETAIL));
    renderDialog();

    submitForm();

    expect((await screen.findByTestId("portal-slug-error")).textContent).toBe(SLUG_CONFLICT_DETAIL);
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
    expect(screen.queryByTestId("portal-slug-error")).toBeNull();
  });

  it("reports an unreadable mapping conflict at dialog level, not on the slug field", async () => {
    mocks.mutateAsync.mockRejectedValue(conflict(MAPPING_CONFLICT_DETAIL));
    renderDialog();

    submitForm();

    expect(await screen.findByText(/already has a customer portal/i)).toBeDefined();
    // No slug can win a mapping conflict held by a portal this workspace cannot
    // read, so the field carries no error.
    expect(screen.queryByTestId("portal-slug-error")).toBeNull();
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("recovers into the workspace's own portal on a mapping conflict", async () => {
    // The server checks the slug before the mapping, so an operator who renamed
    // after a slug conflict lands here holding a portal they already own.
    mocks.mutateAsync.mockRejectedValue(conflict(MAPPING_CONFLICT_DETAIL));
    mocks.getPortalByMapping.mockResolvedValue({ id: "portal_123", slug: "some-other-slug" });
    renderDialog();

    submitForm();

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(mocks.setQueryData).toHaveBeenCalledWith(["portal", "ks_123"], {
      found: true,
      portal: { id: "portal_123", slug: "some-other-slug" },
    });
    expect(screen.queryByText(/already has a customer portal/i)).toBeNull();
  });

  it("recovers from the unique-index conflict when the re-read finds this portal", async () => {
    mocks.mutateAsync.mockRejectedValue(conflict(AMBIGUOUS_CONFLICT_DETAIL));
    mocks.getPortalByMapping.mockResolvedValue({ id: "portal_123", slug: "payments-api" });
    renderDialog();

    submitForm();

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(mocks.setQueryData).toHaveBeenCalledWith(["portal", "ks_123"], {
      found: true,
      portal: { id: "portal_123", slug: "payments-api" },
    });
    expect(screen.queryByTestId("portal-slug-error")).toBeNull();
  });

  it("reports the unique-index conflict on the field when no portal exists", async () => {
    mocks.mutateAsync.mockRejectedValue(conflict(AMBIGUOUS_CONFLICT_DETAIL));
    renderDialog();

    submitForm();

    expect((await screen.findByTestId("portal-slug-error")).textContent).toBe(
      AMBIGUOUS_CONFLICT_DETAIL,
    );
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("does not send the operator to rename when the re-read itself fails", async () => {
    mocks.mutateAsync.mockRejectedValue(conflict(SLUG_CONFLICT_DETAIL));
    mocks.getPortalByMapping.mockRejectedValue(new Error("network down"));
    renderDialog();

    submitForm();

    // The create may well have landed, so the dialog says so rather than
    // claiming a slug collision it could not confirm.
    expect(
      await screen.findByText(/couldn't confirm whether the portal was created/i),
    ).toBeDefined();
    expect(screen.queryByTestId("portal-slug-error")).toBeNull();
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("keeps a slug conflict when the re-read finds a portal with a different slug", async () => {
    mocks.mutateAsync.mockRejectedValue(conflict(SLUG_CONFLICT_DETAIL));
    // A pre-existing keyspace portal plus an unrelated portal holding the slug:
    // adopting this row would close the dialog on a slug that is not live.
    mocks.getPortalByMapping.mockResolvedValue({ id: "portal_999", slug: "something-else" });
    renderDialog();

    submitForm();

    expect((await screen.findByTestId("portal-slug-error")).textContent).toBe(SLUG_CONFLICT_DETAIL);
    expect(mocks.setQueryData).not.toHaveBeenCalled();
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
