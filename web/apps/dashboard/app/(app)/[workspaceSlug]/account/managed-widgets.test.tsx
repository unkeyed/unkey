import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ManagedUserWidgets, ManagedUsersWidget } from "@unkey/workos-widgets";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

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

  it("does not expose whether an email already belongs to another account", async () => {
    const tokenPayload = btoa(
      JSON.stringify({ permissions: [], exp: Math.floor(Date.now() / 1000) + 3600 }),
    );
    const getAccessToken = vi
      .fn<[], Promise<string>>()
      .mockResolvedValue(`header.${tokenPayload}.sig`);
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = new URL(input instanceof Request ? input.url : input.toString());

      if (url.pathname.endsWith("/me")) {
        return Response.json({
          id: "user_123",
          email: "owner@example.com",
          firstName: "Test",
          lastName: "User",
          oauthProfiles: null,
        });
      }

      if (url.pathname.endsWith("/authentication-information")) {
        return Response.json({
          data: {
            verificationMethods: {
              Password: {
                provider: "Password",
                isSetUp: true,
                isCurrentSession: true,
              },
            },
            passwordSettings: {
              isPasswordNumberRequired: false,
              isPasswordPwnedRequired: false,
              isPasswordSymbolRequired: false,
              isPasswordUppercaseRequired: false,
              passwordMinimumLength: 8,
              passwordMinimumStrength: 1,
            },
          },
        });
      }

      if (url.pathname.endsWith("/send-verification")) {
        return Response.json({
          authenticationChallenge: "challenge_123",
          type: "EmailVerification",
        });
      }

      if (url.pathname.endsWith("/verify")) {
        return Response.json({
          elevatedAccessToken: "elevated_token",
          expiresAt: new Date(Date.now() + 60_000).toISOString(),
        });
      }

      if (url.pathname.endsWith("/send-email-change")) {
        return Response.json(
          { code: "email_not_available", message: "This email is not available." },
          { status: 422, statusText: "Unprocessable Entity" },
        );
      }

      throw new Error(`Unexpected WorkOS request: ${init?.method ?? "GET"} ${url.pathname}`);
    });

    const onMutationError = vi.fn((error: unknown) => {
      if (error instanceof Error && error.message === "This email is not available.") {
        error.message =
          "We couldn't update your email. Try again or contact support if the problem continues.";
      }
    });

    render(
      <ManagedUserWidgets getAccessToken={getAccessToken} onMutationError={onMutationError} />,
    );

    await screen.findByText("owner@example.com");
    const emailEditButton = screen.getAllByRole("button", { name: "Edit" }).at(1);
    if (!emailEditButton) {
      throw new Error("Email edit button was not rendered");
    }
    fireEvent.click(emailEditButton);
    fireEvent.click(await screen.findByRole("button", { name: "Send verification code" }));

    const otpInputs = await screen.findAllByRole("textbox");
    for (const [index, input] of otpInputs.entries()) {
      fireEvent.change(input, { target: { value: String(index + 1) } });
    }

    await screen.findByRole("heading", { name: "Change email address" });
    fireEvent.change(screen.getByRole("textbox", { name: "New email address" }), {
      target: { value: "existing@example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Send verification code" }));

    expect(
      await screen.findByText(
        "We couldn't update your email. Try again or contact support if the problem continues.",
      ),
    ).toBeTruthy();
    expect(screen.queryByText("This email is not available.")).toBeNull();
    expect(onMutationError).toHaveBeenCalledWith(expect.any(Error));
    await waitFor(() =>
      expect(fetchSpy).toHaveBeenCalledWith(
        expect.objectContaining({ pathname: expect.stringContaining("/send-email-change") }),
        expect.anything(),
      ),
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
