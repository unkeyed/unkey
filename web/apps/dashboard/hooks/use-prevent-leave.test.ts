import { act, renderHook } from "@testing-library/react";
import type { Route } from "next";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { isLeaveSentinel, usePreventLeave } from "./use-prevent-leave";

const router = { replace: vi.fn(), push: vi.fn() };
vi.mock("next/navigation", () => ({ useRouter: () => router }));

const FALLBACK = "/fallback" as Route;

function popstate(): Promise<void> {
  return new Promise((resolve) => {
    window.addEventListener("popstate", () => resolve(), { once: true });
  });
}

function clickLink(href: string, init: MouseEventInit = {}): MouseEvent {
  const anchor = document.createElement("a");
  anchor.href = href;
  document.body.append(anchor);
  const event = new MouseEvent("click", { bubbles: true, cancelable: true, button: 0, ...init });
  anchor.dispatchEvent(event);
  anchor.remove();
  return event;
}

describe("usePreventLeave", () => {
  beforeEach(() => {
    router.replace.mockClear();
    router.push.mockClear();
    window.history.replaceState(null, "", "/");
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("pushes a sentinel while enabled and pops it again when disabled", async () => {
    const { rerender, unmount } = renderHook(({ enabled }) => usePreventLeave(enabled, FALLBACK), {
      initialProps: { enabled: true },
    });
    expect(isLeaveSentinel(window.history.state)).toBe(true);

    const popped = popstate();
    rerender({ enabled: false });
    await act(() => popped);
    expect(isLeaveSentinel(window.history.state)).toBe(false);
    unmount();
  });

  it("asks on the first Back after the guard turns off and on again", async () => {
    const { result, rerender, unmount } = renderHook(
      ({ enabled }) => usePreventLeave(enabled, FALLBACK),
      { initialProps: { enabled: true } },
    );

    const cleared = popstate();
    rerender({ enabled: false });
    await act(() => cleared);
    rerender({ enabled: true });
    expect(isLeaveSentinel(window.history.state)).toBe(true);

    const back = popstate();
    window.history.back();
    await act(() => back);
    expect(result.current.leavePrompt.open).toBe(true);
    unmount();
  });

  it("asks before following an in-app link and replaces the sentinel on discard", async () => {
    const { result, unmount } = renderHook(() => usePreventLeave(true, FALLBACK));

    let event: MouseEvent | undefined;
    act(() => {
      event = clickLink("/other?x=1");
    });
    expect(event?.defaultPrevented).toBe(true);
    expect(result.current.leavePrompt.open).toBe(true);

    act(() => result.current.leavePrompt.onDiscard());
    expect(router.replace).toHaveBeenCalledWith("/other?x=1");
    expect(result.current.leavePrompt.open).toBe(false);
    unmount();
  });

  it("asks on Back and goes to the fallback when there is nothing to go back to", async () => {
    vi.useFakeTimers();
    const back = vi.spyOn(window.history, "back").mockImplementation(() => {});
    const { result, unmount } = renderHook(() => usePreventLeave(true, FALLBACK));

    act(() => {
      window.dispatchEvent(new PopStateEvent("popstate", { state: null }));
    });
    expect(result.current.leavePrompt.open).toBe(true);

    act(() => result.current.leavePrompt.onDiscard());
    act(() => {
      vi.runAllTimers();
    });
    expect(back).toHaveBeenCalledTimes(1);
    expect(router.replace).toHaveBeenCalledWith(FALLBACK);
    back.mockRestore();
    unmount();
  });
});
