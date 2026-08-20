import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { Portal } from "@unkey/api/models/components";
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from "react";
import { forwardRef } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PortalConfig } from "./portal-config";

const SLUG_CONFLICT_DETAIL = "That slug is already in use. Choose a different slug.";

/** Both portal 409s arrive under the same code, so only `detail` separates them. */
class ApiError extends Error {
  constructor(readonly detail: string) {
    super(detail);
  }
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

vi.mock("@/lib/unkey-client", () => ({
  getErrorMessage: (error: unknown) =>
    error instanceof ApiError ? error.detail : "Something went wrong",
}));

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
  return {
    Input,
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
      title,
      children,
      footer,
    }: {
      isOpen: boolean;
      title?: string;
      children?: ReactNode;
      footer?: ReactNode;
    }) =>
      isOpen ? (
        <div>
          <span>{title}</span>
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
  enabled: true,
  mapping: { type: "keyspace", id: "ks_123" },
  createdAt: 0,
};

function renderConfig(overrides?: Partial<Portal>) {
  return render(
    <PortalConfig
      portal={{ ...portal, ...overrides }}
      keyAuthId="ks_123"
      resourceName="Acme API"
    />,
  );
}

function input(label: string): HTMLInputElement {
  const element = screen.getByLabelText(label);
  if (!(element instanceof HTMLInputElement)) {
    throw new Error(`${label} is not an input`);
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

  it("warns before saving once the slug is edited", async () => {
    renderConfig();

    expect(screen.queryByText(/breaks every/i)).toBeNull();

    fireEvent.change(input("Slug"), { target: { value: "acme-two" } });

    expect(await screen.findByText(/breaks every/i)).toBeDefined();
  });

  it("puts a slug conflict on the field rather than in a toast", async () => {
    mocks.updateMutation.mutateAsync.mockRejectedValue(new ApiError(SLUG_CONFLICT_DETAIL));
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
    fireEvent.click(screen.getAllByRole("button", { name: "Disable portal" })[1]);

    await waitFor(() =>
      expect(mocks.updateMutation.mutate).toHaveBeenCalledWith({
        portal: "portal_123",
        enabled: false,
      }),
    );
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
});
