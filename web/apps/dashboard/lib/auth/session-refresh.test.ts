import type { AuthkitResponse } from "@workos-inc/authkit-nextjs";
import { describe, expect, it, vi } from "vitest";
import { coalesceSessionRefresh } from "./session-refresh";

function response(): AuthkitResponse {
  return {
    session: { user: null },
    headers: new Headers(),
    authorizationUrl: "https://auth.example.com",
  };
}

function gate() {
  let open: () => void = () => {};
  const held = new Promise<void>((resolve) => {
    open = () => resolve();
  });
  return { held, open: () => open() };
}

describe("coalesceSessionRefresh", () => {
  it("runs one refresh for concurrent requests sharing a session cookie", async () => {
    const held = gate();
    const refresh = vi.fn(async () => {
      await held.held;
      return response();
    });

    const first = coalesceSessionRefresh("session-a", refresh);
    const second = coalesceSessionRefresh("session-a", refresh);
    held.open();

    expect(await first).toBe(await second);
    expect(refresh).toHaveBeenCalledOnce();
  });

  it("runs a refresh per distinct session cookie", async () => {
    const refresh = vi.fn(async () => response());

    await Promise.all([
      coalesceSessionRefresh("session-b", refresh),
      coalesceSessionRefresh("session-c", refresh),
    ]);

    expect(refresh).toHaveBeenCalledTimes(2);
  });

  it("does not share refreshes for requests without a session", async () => {
    const held = gate();
    const refresh = vi.fn(async () => {
      await held.held;
      return response();
    });

    const first = coalesceSessionRefresh(undefined, refresh);
    const second = coalesceSessionRefresh(undefined, refresh);
    held.open();

    await Promise.all([first, second]);
    expect(refresh).toHaveBeenCalledTimes(2);
  });

  it("starts a new refresh once the previous one settled", async () => {
    const refresh = vi.fn(async () => response());

    await coalesceSessionRefresh("session-d", refresh);
    await coalesceSessionRefresh("session-d", refresh);

    expect(refresh).toHaveBeenCalledTimes(2);
  });

  it("does not retain a failed refresh", async () => {
    const failing = vi.fn(async () => {
      throw new Error("workos unavailable");
    });

    await expect(coalesceSessionRefresh("session-e", failing)).rejects.toThrow(
      "workos unavailable",
    );
    await expect(coalesceSessionRefresh("session-e", failing)).rejects.toThrow(
      "workos unavailable",
    );

    expect(failing).toHaveBeenCalledTimes(2);
  });
});
