import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AuthenticationErrorContent } from "./error-content";

describe("AuthenticationErrorPage", () => {
  it("shows a generic retry and moves focus to the error heading", async () => {
    render(<AuthenticationErrorContent />);

    const heading = screen.getByRole("heading", { name: "We could not sign you in" });
    expect(screen.getByRole("link", { name: "Sign in again" }).getAttribute("href")).toBe(
      "/auth/sign-in",
    );
    expect(screen.queryByText(/reason=/i)).toBeNull();
    await waitFor(() => expect(document.activeElement).toBe(heading));
  });
});
