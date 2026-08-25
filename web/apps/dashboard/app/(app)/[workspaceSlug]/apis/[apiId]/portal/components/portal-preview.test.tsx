import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { PortalPreview } from "./portal-preview";

afterEach(cleanup);

function renderPreview(overrides?: {
  logoUrl?: string;
  primaryColor?: string;
  displayName?: string;
}) {
  const { container } = render(
    <PortalPreview
      displayName={overrides?.displayName ?? "Acme Inc"}
      branding={{
        logoUrl: overrides?.logoUrl ?? "",
        primaryColor: overrides?.primaryColor ?? "#18181B",
      }}
    />,
  );
  return container;
}

describe("PortalPreview", () => {
  it("shows the display name in the brand bar, matching what the live portal renders", () => {
    renderPreview({ displayName: "Acme Inc" });
    expect(screen.getByText("Acme Inc")).toBeTruthy();
  });

  // The portal ships light-only. `light` re-declares the palette on this
  // subtree, so the chrome keeps its light values inside a dark dashboard.
  // Without it the `gray-*` tokens below resolve to their `.dark` values.
  it("pins itself to the light palette", () => {
    const container = renderPreview();
    const root = container.firstElementChild;
    expect(root?.classList.contains("light")).toBe(true);
  });

  // The dashboard's Tailwind config replaces the default palette, so classes
  // from it compile to nothing and render invisible.
  it("styles its chrome from the workspace palette, not Tailwind defaults", () => {
    const container = renderPreview();
    const unresolvable = container.querySelectorAll(
      '[class*="white"], [class*="neutral-"], [class*="slate-"], [class*="zinc-"], [class*="stone-"]',
    );
    expect(unresolvable.length).toBe(0);
  });

  it("renders the window dots and row skeletons with a fill", () => {
    const container = renderPreview();
    const filled = container.querySelectorAll('[class*="bg-gray-"]');
    // 3 window dots + the address bar + heading/sub skeletons + 4 rows.
    expect(filled.length).toBeGreaterThanOrEqual(12);
  });

  it("uses dark foreground on a light brand color", () => {
    renderPreview({ primaryColor: "#FFFF00", displayName: "bright" });
    const label = screen.getByText("bright");
    expect(label instanceof HTMLElement && label.style.color).toBe("rgb(10, 10, 10)");
  });

  it("uses light foreground on a dark brand color", () => {
    renderPreview({ primaryColor: "#18181B", displayName: "dim" });
    const label = screen.getByText("dim");
    expect(label instanceof HTMLElement && label.style.color).toBe("rgb(255, 255, 255)");
  });

  it("renders a valid https logo", () => {
    const container = renderPreview({ logoUrl: "https://cdn.example.com/logo.png" });
    expect(container.querySelector("img")).toBeTruthy();
  });

  // The gate lives at the sink, so a caller that forgets to validate cannot turn
  // the preview into an arbitrary-URL fetcher.
  it.each([
    ["a non-https scheme", "http://example.com/logo.png"],
    ["a javascript: url", "javascript:alert(1)"],
    ["an unparseable string", "not a url"],
  ])("does not render %s as an image", (_label, logoUrl) => {
    const container = renderPreview({ logoUrl });
    expect(container.querySelector("img")).toBeNull();
  });
});
