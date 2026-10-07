import { cleanup, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { IntegrateDialog } from "./integrate-dialog";

// Every tab's panel is rendered so one pass can inspect all three snippets;
// the real Tabs only mount the active panel.
vi.mock("@unkey/ui", () => ({
  DialogContainer: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  Code: ({ children }: { children?: ReactNode }) => <pre>{children}</pre>,
  CopyButton: ({ value }: { value: string }) => <button type="button">{`copy:${value}`}</button>,
  Tabs: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children, value }: { children?: ReactNode; value: string }) => (
    <button type="button" data-testid={`trigger-${value}`}>
      {children}
    </button>
  ),
  TabsContent: ({ children, value }: { children?: ReactNode; value: string }) => (
    <div data-testid={`panel-${value}`}>{children}</div>
  ),
}));

function renderDialog() {
  return render(<IntegrateDialog slug="acme-portal" isOpen onOpenChange={() => {}} />);
}

describe("IntegrateDialog", () => {
  // Vitest globals are off here, so testing-library does not auto-unmount.
  beforeEach(cleanup);

  it("emits a cURL snippet matching the shipped createSession schema", () => {
    renderDialog();

    const curl = screen.getByTestId("panel-curl").textContent ?? "";

    expect(curl).toContain("portal.createSession");
    expect(curl).toContain('"portal": "acme-portal"');
    expect(curl).toContain('"externalId"');
    expect(curl).toContain('"scopes"');
    expect(curl).toContain("keys:read");
    expect(curl).toContain("keys:reroll");
    expect(curl).toContain("analytics:read");
    // The prototype's field names would 400 against the real endpoint.
    expect(curl).not.toContain("permissions");
    expect(curl).not.toContain("portalId");
    expect(curl).not.toContain("keys.read");
  });

  it("renders a snippet carrying the slug on each of the three language tabs", () => {
    renderDialog();

    for (const language of ["curl", "ts", "go"]) {
      expect(screen.getByTestId(`trigger-${language}`)).toBeDefined();
      const snippet = screen.getByTestId(`panel-${language}`).textContent ?? "";
      expect(snippet).toContain("acme-portal");
      expect(snippet).not.toContain("permissions");
      expect(snippet).not.toContain("portalId");
    }
  });

  // A snippet naming a scope outside this set hands out a payload createSession refuses.
  it("offers exactly the scopes createSession accepts", () => {
    renderDialog();

    for (const language of ["curl", "ts"]) {
      const snippet = screen.getByTestId(`panel-${language}`).textContent ?? "";
      expect(snippet).toContain("keys:read");
      expect(snippet).toContain("keys:reroll");
      expect(snippet).toContain("analytics:read");
      expect(snippet).not.toContain("keys:create");
    }

    const go = screen.getByTestId("panel-go").textContent ?? "";
    expect(go).toContain("ScopeKeysRead");
    expect(go).toContain("ScopeKeysReroll");
    expect(go).toContain("ScopeAnalyticsRead");
    expect(go).not.toContain("ScopeKeysCreate");
  });

  // Prose spans <code> children, so it is read off the container rather than by text node.
  it("explains what analytics:read grants and that it needs keys:read alongside it", () => {
    const { container } = renderDialog();
    const prose = container.textContent ?? "";

    expect(prose).toContain("analytics:read shows them their own key usage over time");
    expect(prose).toContain("each require keys:read in the same session");
    expect(prose).toContain("analytics:read needs read_analytics");
  });

  it("documents returnUrl with the trust caveat that stops an open redirect", () => {
    renderDialog();

    expect(screen.getByText(/never take it from the incoming request/i)).toBeDefined();
    expect(screen.getByText(/open redirect/i)).toBeDefined();
  });
});
