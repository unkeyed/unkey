import { cleanup, render, screen } from "@testing-library/react";
import { ManagedUserWidgets, ManagedUsersWidget } from "@unkey/workos-widgets";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";

afterEach(cleanup);

beforeAll(() => {
  vi.stubGlobal(
    "ResizeObserver",
    class ResizeObserver {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
});

afterAll(() => {
  vi.unstubAllGlobals();
});

describe("ManagedUserWidgets", () => {
  it("renders the real profile and security widgets without session management", () => {
    const getAccessToken = vi.fn<[], Promise<string>>().mockResolvedValue("access_token");
    const { container } = render(<ManagedUserWidgets getAccessToken={getAccessToken} />);

    const profileHeading = screen.getByRole("heading", { name: "Profile" });
    const securityHeading = screen.getByRole("heading", { name: "Security" });
    expect(profileHeading).toBeTruthy();
    expect(securityHeading).toBeTruthy();
    expect(container.querySelector('[data-woswidgets-widget-id="user-profile"]')).toBeTruthy();
    expect(container.querySelector('[data-woswidgets-widget-id="user-security"]')).toBeTruthy();
    expect(container.querySelector('[data-woswidgets-widget-id="user-sessions"]')).toBeNull();
    expect(profileHeading.className).toContain("m-0");
    expect(profileHeading.parentElement?.className).toContain("gap-3");
    expect(securityHeading.parentElement?.className).toContain("gap-1");
    expect(profileHeading.parentElement?.parentElement?.className).toContain("gap-8");

    const widgetTheme = container.querySelector(".unkey-workos-widgets");
    expect(widgetTheme?.getAttribute("data-accent-color")).toBe("gray");
    expect(widgetTheme?.getAttribute("data-gray-color")).toBe("gray");
    expect(widgetTheme?.getAttribute("data-has-background")).toBe("false");
    expect(widgetTheme?.getAttribute("data-panel-background")).toBe("solid");
    expect(widgetTheme?.getAttribute("data-radius")).toBe("medium");
    expect(widgetTheme?.getAttribute("data-scaling")).toBe("100%");
    expect(widgetTheme?.getAttribute("style")).toContain(
      "--default-font-family: var(--font-geist-sans)",
    );
  });
});

describe("ManagedUsersWidget", () => {
  it("renders the WorkOS user-management widget with the Unkey theme", () => {
    const getAccessToken = vi.fn<[], Promise<string>>().mockResolvedValue("access_token");
    const { container } = render(<ManagedUsersWidget getAccessToken={getAccessToken} />);

    expect(container.querySelector('[data-woswidgets-widget-id="users-management"]')).toBeTruthy();

    const widgetTheme = container.querySelector(".unkey-workos-widgets");
    expect(widgetTheme?.getAttribute("data-accent-color")).toBe("gray");
    expect(widgetTheme?.getAttribute("data-gray-color")).toBe("gray");
    expect(widgetTheme?.getAttribute("data-has-background")).toBe("false");
  });
});
