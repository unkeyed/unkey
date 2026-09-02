import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { Portal } from "@unkey/api/models/components";
import { ConflictErrorResponse } from "@unkey/api/models/errors";
import type { ButtonHTMLAttributes, HTMLAttributes, InputHTMLAttributes, ReactNode } from "react";
import { forwardRef } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PortalConfig } from "./portal-config";

const SLUG_CONFLICT_DETAIL = "That slug is already in use. Choose a different slug.";

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
  updateMutation: { mutate: vi.fn(), mutateAsync: vi.fn(), isLoading: false },
  deleteMutation: { mutate: vi.fn(), mutateAsync: vi.fn(), isLoading: false },
  toastSuccess: vi.fn(),
}));

vi.mock("@/lib/portal/use-portal", () => ({
  useUpdatePortal: () => mocks.updateMutation,
  useDeletePortal: () => mocks.deleteMutation,
}));

vi.mock("@/lib/unkey-client", async () => {
  const { ConflictErrorResponse: Conflict } = await import("@unkey/api/models/errors");
  return {
    getErrorMessage: (error: unknown) =>
      error instanceof Conflict ? error.error.detail : "Something went wrong",
  };
});

vi.mock("./portal-preview", () => ({
  PortalPreview: ({ slug }: { slug: string }) => <div data-testid="preview">{slug}</div>,
}));

vi.mock("@unkey/icons", () => ({
  TriangleWarning2: () => null,
}));

vi.mock("@unkey/ui", () => {
  const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
    (props, ref) => <input ref={ref} {...props} />,
  );
  const InputGroupInput = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
    (props, ref) => <input ref={ref} {...props} />,
  );
  return {
    Input,
    InputGroup: ({
      children,
      variant: _variant,
      ...props
    }: HTMLAttributes<HTMLDivElement> & { variant?: string }) => <div {...props}>{children}</div>,
    InputGroupInput,
    InputGroupAddon: ({ children, align: _align }: { children?: ReactNode; align?: string }) => (
      <span>{children}</span>
    ),
    FormLabel: ({
      label,
      htmlFor,
    }: {
      label?: string;
      htmlFor?: string;
      hasError?: boolean;
      tooltipContent?: ReactNode;
    }) => <label htmlFor={htmlFor}>{label}</label>,
    FormDescription: ({ error }: { error?: string; descriptionId?: string; errorId?: string }) =>
      error ? <span data-testid="slug-error">{error}</span> : null,
    FormField: ({
      label,
      error,
      children,
    }: {
      label?: string;
      description?: ReactNode;
      descriptionPosition?: string;
      error?: string;
      children: (field: {
        id: string;
        describedBy: string | undefined;
        invalid: boolean;
        variant: "error" | undefined;
      }) => ReactNode;
    }) => (
      <fieldset>
        <label htmlFor="form-field">{label}</label>
        {children({
          id: "form-field",
          describedBy: error ? "form-field-error" : undefined,
          invalid: Boolean(error),
          variant: error ? "error" : undefined,
        })}
        {error ? <span data-testid="slug-error">{error}</span> : null}
      </fieldset>
    ),
    Button: ({
      children,
      loading: _loading,
      loadingLabel: _loadingLabel,
      variant: _variant,
      size: _size,
      color: _color,
      ...props
    }: ButtonHTMLAttributes<HTMLButtonElement> & {
      loading?: boolean;
      loadingLabel?: string;
      variant?: string;
      size?: string;
      color?: string;
    }) => <button {...props}>{children}</button>,
    CopyButton: ({ value }: { value: string }) => (
      <button type="button" aria-label={`Copy ${value}`} />
    ),
    // Honouring `isOpen` keeps the disable and delete dialogs from colliding.
    DialogContainer: ({
      isOpen,
      onOpenChange,
      title,
      children,
      footer,
    }: {
      isOpen: boolean;
      onOpenChange?: (open: boolean) => void;
      title?: string;
      children?: ReactNode;
      footer?: ReactNode;
    }) =>
      isOpen ? (
        <div>
          <span>{title}</span>
          {/* Stands in for the real dialog's dismiss affordances. */}
          <button
            type="button"
            aria-label={`Close ${title}`}
            onClick={() => onOpenChange?.(false)}
          />
          {children}
          {footer}
        </div>
      ) : null,
    FormInput: forwardRef<
      HTMLInputElement,
      InputHTMLAttributes<HTMLInputElement> & {
        label?: string;
        error?: string;
        description?: string;
        descriptionPosition?: string;
      }
    >(({ label, error, description: _d, descriptionPosition: _dp, ...props }, ref) => (
      <span>
        <label htmlFor={props.name}>{label}</label>
        <input id={props.name} ref={ref} {...props} />
        {error ? <span data-testid={`${props.name}-error`}>{error}</span> : null}
      </span>
    )),
    SettingsDangerZone: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
    SettingsZoneRow: ({
      title,
      description,
      action,
    }: {
      title: ReactNode;
      description: ReactNode;
      action: { label: string; onClick: () => void };
    }) => (
      <div>
        <p>{title}</p>
        <p>{description}</p>
        <button type="button" onClick={action.onClick}>
          {action.label}
        </button>
      </div>
    ),
    toast: { success: mocks.toastSuccess, error: vi.fn() },
  };
});

const portal: Portal = {
  id: "portal_123",
  slug: "acme",
  displayName: "Acme",
  enabled: true,
  keyspaceId: "ks_123",
  createdAt: 0,
};

function renderConfig(overrides?: Partial<Portal>) {
  return render(<PortalConfig portal={{ ...portal, ...overrides }} keyAuthId="ks_123" />);
}

function input(label: string): HTMLInputElement {
  const element = screen.getByLabelText(label);
  if (!(element instanceof HTMLInputElement)) {
    throw new Error(`${label} is not an input`);
  }
  return element;
}

function form(within: HTMLElement): HTMLFormElement {
  const element = within.closest("form");
  if (!element) {
    throw new Error("no enclosing form");
  }
  return element;
}

function button(name: string, index = 0): HTMLButtonElement {
  const element = screen.getAllByRole("button", { name })[index];
  if (!(element instanceof HTMLButtonElement)) {
    throw new Error(`${name} is not a button`);
  }
  return element;
}

function saveButton(): HTMLButtonElement {
  const element = screen.getByRole("button", { name: "Save" });
  if (!(element instanceof HTMLButtonElement)) {
    throw new Error("Save is not a button");
  }
  return element;
}

describe("PortalConfig", () => {
  // Vitest globals are off here, so testing-library does not auto-unmount.
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    mocks.updateMutation.mutateAsync.mockResolvedValue({ portalId: "portal_123" });
    mocks.deleteMutation.mutateAsync.mockResolvedValue({});
  });

  it("leaves branding empty and save disabled for a portal with no branding", () => {
    renderConfig({ branding: undefined });

    expect(input("Logo URL").value).toBe("");
    expect(input("Primary color").value).toBe("");
    expect(saveButton().disabled).toBe(true);
  });

  it("carries no name field", () => {
    renderConfig();

    expect(screen.queryByLabelText("Name")).toBeNull();
  });

  it("sends only the changed key when the primary color is edited", async () => {
    renderConfig({ branding: { logoUrl: "https://cdn.example.com/logo.png" } });

    fireEvent.click(screen.getByRole("button", { name: "Use #7C3AED" }));
    await waitFor(() => expect(saveButton().disabled).toBe(false));

    fireEvent.click(saveButton());

    await waitFor(() => expect(mocks.updateMutation.mutateAsync).toHaveBeenCalledTimes(1));
    expect(mocks.updateMutation.mutateAsync).toHaveBeenCalledWith({
      portal: "portal_123",
      primaryColor: "#7C3AED",
    });
  });

  it("clears a logo by sending null rather than an empty string", async () => {
    renderConfig({ branding: { logoUrl: "https://cdn.example.com/logo.png" } });

    fireEvent.change(input("Logo URL"), { target: { value: "" } });
    await waitFor(() => expect(saveButton().disabled).toBe(false));

    fireEvent.click(saveButton());

    await waitFor(() => expect(mocks.updateMutation.mutateAsync).toHaveBeenCalledTimes(1));
    expect(mocks.updateMutation.mutateAsync).toHaveBeenCalledWith({
      portal: "portal_123",
      logoUrl: null,
    });
  });

  it("does not clear untouched branding when the portal prop refreshes mid-edit", async () => {
    // A teammate sets a logo while this page is open; the query refetch swaps in
    // a fresh portal object, but the form keeps the snapshot it mounted with.
    const { rerender } = renderConfig({ branding: undefined });

    rerender(
      <PortalConfig
        portal={{ ...portal, branding: { logoUrl: "https://cdn.example.com/theirs.png" } }}
        keyAuthId="ks_123"
      />,
    );

    fireEvent.change(input("Slug"), { target: { value: "acme-two" } });
    await waitFor(() => expect(saveButton().disabled).toBe(false));
    fireEvent.click(saveButton());

    await waitFor(() => expect(mocks.updateMutation.mutateAsync).toHaveBeenCalledTimes(1));
    expect(mocks.updateMutation.mutateAsync).toHaveBeenCalledWith({
      portal: "portal_123",
      slug: "acme-two",
    });
  });

  it("warns before saving once the slug is edited", async () => {
    renderConfig();

    expect(screen.queryByText(/breaks every/i)).toBeNull();

    fireEvent.change(input("Slug"), { target: { value: "acme-two" } });

    expect(await screen.findByText(/breaks every/i)).toBeDefined();
  });

  it("puts a slug conflict on the field rather than in a toast", async () => {
    mocks.updateMutation.mutateAsync.mockRejectedValue(conflict(SLUG_CONFLICT_DETAIL));
    renderConfig();

    fireEvent.change(input("Slug"), { target: { value: "taken" } });
    await waitFor(() => expect(saveButton().disabled).toBe(false));
    fireEvent.click(saveButton());

    expect((await screen.findByTestId("slug-error")).textContent).toBe(SLUG_CONFLICT_DETAIL);
    expect(mocks.toastSuccess).not.toHaveBeenCalled();
  });

  it("disables the portal without deleting it", async () => {
    renderConfig();

    fireEvent.click(screen.getByRole("button", { name: "Disable portal" }));
    fireEvent.click(button("Disable portal", 1));

    // Awaited, not fire-and-forget, so the dialog can paint its loading state.
    await waitFor(() =>
      expect(mocks.updateMutation.mutateAsync).toHaveBeenCalledWith({
        portal: "portal_123",
        enabled: false,
      }),
    );
    await waitFor(() => expect(screen.queryByText("Disable customer portal?")).toBeNull());
    expect(mocks.deleteMutation.mutate).not.toHaveBeenCalled();
    expect(mocks.deleteMutation.mutateAsync).not.toHaveBeenCalled();
  });

  it("keeps delete disabled until the portal slug is typed exactly", async () => {
    renderConfig();

    fireEvent.click(screen.getByRole("button", { name: "Delete portal" }));
    const confirm = screen.getAllByRole("button", { name: "Delete portal" })[1];
    if (!(confirm instanceof HTMLButtonElement)) {
      throw new Error("delete confirmation is not a button");
    }
    expect(confirm.disabled).toBe(true);

    fireEvent.change(input("Portal slug confirmation"), { target: { value: "acm" } });
    await waitFor(() => expect(confirm.disabled).toBe(true));

    fireEvent.change(input("Portal slug confirmation"), { target: { value: "acme" } });
    await waitFor(() => expect(confirm.disabled).toBe(false));

    fireEvent.click(confirm);
    await waitFor(() =>
      expect(mocks.deleteMutation.mutateAsync).toHaveBeenCalledWith({ portal: "portal_123" }),
    );
  });
  it("issues no request when a save touches nothing", async () => {
    renderConfig();

    // Edited and reverted: react-hook-form drops the field from `dirtyFields`.
    fireEvent.change(input("Slug"), { target: { value: "acme-two" } });
    await waitFor(() => expect(saveButton().disabled).toBe(false));
    fireEvent.change(input("Slug"), { target: { value: "acme" } });
    await waitFor(() => expect(saveButton().disabled).toBe(true));

    fireEvent.submit(form(input("Slug")));

    await waitFor(() => expect(mocks.updateMutation.mutateAsync).not.toHaveBeenCalled());
  });

  it("ignores a delete submit whose confirmation does not match", async () => {
    renderConfig();

    fireEvent.click(screen.getByRole("button", { name: "Delete portal" }));
    fireEvent.change(input("Portal slug confirmation"), { target: { value: "acm" } });

    fireEvent.submit(form(input("Portal slug confirmation")));

    await waitFor(() => expect(mocks.deleteMutation.mutateAsync).not.toHaveBeenCalled());
  });

  it("clears the delete confirmation when the dialog is closed and reopened", async () => {
    renderConfig();

    fireEvent.click(screen.getByRole("button", { name: "Delete portal" }));
    fireEvent.change(input("Portal slug confirmation"), { target: { value: "acme" } });
    await waitFor(() => expect(button("Delete portal", 1).disabled).toBe(false));

    fireEvent.click(screen.getByRole("button", { name: "Close Delete customer portal" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete portal" }));

    expect(input("Portal slug confirmation").value).toBe("");
    expect(button("Delete portal", 1).disabled).toBe(true);
  });
});
