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

  // The portal ships light-only. If the chrome is built from `gray-*`/`grayA-*`
  // tokens it inverts with the operator's dashboard theme and previews a portal
  // their users never see.
  it("paints its chrome in fixed light values, not theme-reactive tokens", () => {
    const container = renderPreview();
    const themed = container.querySelectorAll(
      '[class*="bg-gray-"], [class*="bg-grayA-"], [class*="border-grayA-"], [class*="text-gray-"], [class*="bg-background"]',
    );
    expect(themed.length).toBe(0);
  });

  // A class-name check alone cannot catch this: the dashboard's Tailwind config
  // replaces the default palette, so `bg-white`/`bg-neutral-300` compile to
  // nothing and render invisible. Assert the surface actually carries a color.
  it("renders a real light surface rather than an unresolved class", () => {
    const container = renderPreview();
    const root = container.firstElementChild;
    expect(root instanceof HTMLElement && root.style.backgroundColor).toBe("rgb(255, 255, 255)");
  });

  it("renders the window dots and row skeletons with a visible fill", () => {
    const container = renderPreview();
    const filled = [...container.querySelectorAll("div, span")].filter(
      (el) => el instanceof HTMLElement && el.style.backgroundColor.startsWith("rgb("),
    );
    // 3 window dots + the address bar + brand bar + heading/sub skeletons + 4 rows.
    expect(filled.length).toBeGreaterThanOrEqual(12);
  });

  it("keeps the mock light for user-agent-styled descendants", () => {
    const container = renderPreview();
    const root = container.firstElementChild;
    expect(root instanceof HTMLElement && root.style.colorScheme).toBe("light");
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
