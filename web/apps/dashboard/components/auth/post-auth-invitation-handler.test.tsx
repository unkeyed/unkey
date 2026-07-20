import { render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  replace: vi.fn(),
  toastError: vi.fn(),
  toastInfo: vi.fn(),
}));

vi.mock("@unkey/ui", () => ({
  toast: { error: mocks.toastError, info: mocks.toastInfo },
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mocks.replace }),
  useSearchParams: () => new URLSearchParams("invitation_token=tok_abc"),
}));

import { PostAuthInvitationHandler } from "./post-auth-invitation-handler";

const GENERIC_ERROR = "We couldn't process your invitation. Ask for a new one.";

function respondWith(status: number, body: unknown) {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ status, json: async () => body } as Response));
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers({ shouldAdvanceTime: true });
  // window.location.reload is not implemented in jsdom.
  Object.defineProperty(window, "location", {
    value: { ...window.location, reload: vi.fn(), pathname: "/apis" },
    writable: true,
  });
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("PostAuthInvitationHandler", () => {
  /**
   * Guarantees a rejected invitation is never silent. This component renders
   * nothing, so before it toasted, a user whose invitation failed saw an
   * ordinary dashboard and no explanation. That was the ENG-3014 symptom.
   */
  it("shows the API's message when the invitation is rejected", async () => {
    respondWith(400, {
      success: false,
      error: "This invitation was sent to a different email address.",
    });

    render(<PostAuthInvitationHandler />);
    await vi.advanceTimersByTimeAsync(600);

    await waitFor(() => {
      expect(mocks.toastError).toHaveBeenCalledWith(
        "This invitation was sent to a different email address.",
        expect.anything(),
      );
    });
  });

  /**
   * Guarantees plumbing text never reaches a user. Only the 400 path carries a
   * message the domain layer vetted; a 500 answers with "Internal server error".
   */
  it("replaces server-error text with the generic message", async () => {
    respondWith(500, { success: false, error: "Internal server error" });

    render(<PostAuthInvitationHandler />);
    await vi.advanceTimersByTimeAsync(600);
    // Exhaust the retry budget.
    await vi.advanceTimersByTimeAsync(5000);

    await waitFor(() => {
      expect(mocks.toastError).toHaveBeenCalledWith(GENERIC_ERROR, expect.anything());
    });
    expect(mocks.toastError).not.toHaveBeenCalledWith("Internal server error", expect.anything());
  });

  /**
   * Guarantees the partial outcome is explained rather than reloaded through.
   * The user is a member but the org is not active, so reloading would drop
   * them into the wrong workspace with no way to understand why.
   */
  it("explains a successful join that could not open the workspace", async () => {
    respondWith(200, { success: true, organizationId: "org_1", switched: false });

    render(<PostAuthInvitationHandler />);
    await vi.advanceTimersByTimeAsync(600);

    await waitFor(() => {
      expect(mocks.toastInfo).toHaveBeenCalled();
    });
    expect(window.location.reload).not.toHaveBeenCalled();
  });

  it("reloads into the new workspace on a complete success", async () => {
    respondWith(200, { success: true, organizationId: "org_1", switched: true });

    render(<PostAuthInvitationHandler />);
    await vi.advanceTimersByTimeAsync(600);
    await vi.advanceTimersByTimeAsync(200);

    await waitFor(() => {
      expect(window.location.reload).toHaveBeenCalled();
    });
    expect(mocks.toastError).not.toHaveBeenCalled();
  });

  /**
   * Guarantees a malformed body cannot put arbitrary text into a toast.
   */
  it("falls back to the generic message when the body is not the expected shape", async () => {
    respondWith(400, "<html>gateway error</html>");

    render(<PostAuthInvitationHandler />);
    await vi.advanceTimersByTimeAsync(600);

    await waitFor(() => {
      expect(mocks.toastError).toHaveBeenCalledWith(GENERIC_ERROR, expect.anything());
    });
  });

  /**
   * Guarantees the invitation is submitted exactly once when the component
   * re-renders while a request is still in flight. The effect's cleanup only
   * cancels a pending timer, and the processed flag is not set until the
   * request settles, so without an in-flight guard the second render schedules
   * a second POST. Acceptance is irreversible on the server: a duplicate is not
   * a wasted round trip but an attempt to consume an already spent token.
   */
  it("submits the invitation only once when re-rendered mid-flight", async () => {
    let settleFetch: (value: unknown) => void = () => {};
    const inFlight = new Promise((resolve) => {
      settleFetch = resolve;
    });
    const fetchMock = vi.fn().mockReturnValue(
      inFlight.then(() => ({
        status: 200,
        json: async () => ({ success: true, organizationId: "org_1", switched: true }),
      })),
    );
    vi.stubGlobal("fetch", fetchMock);

    const { rerender } = render(<PostAuthInvitationHandler />);
    // First attempt is now awaiting the response.
    await vi.advanceTimersByTimeAsync(600);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    // A re-render mid-flight re-runs the effect and schedules another attempt.
    rerender(<PostAuthInvitationHandler />);
    await vi.advanceTimersByTimeAsync(600);

    expect(fetchMock).toHaveBeenCalledTimes(1);

    settleFetch(undefined);
    await vi.advanceTimersByTimeAsync(200);
  });
});
