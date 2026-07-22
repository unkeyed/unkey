import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useControllableState } from "./use-controllable-state";

describe("useControllableState", () => {
  it("uses the controlled prop as the value and lets it win over internal state", () => {
    const onChange = vi.fn();
    const { result, rerender } = renderHook(
      ({ prop }: { prop: boolean }) => useControllableState({ prop, defaultProp: false, onChange }),
      { initialProps: { prop: true } },
    );

    // Controlled prop wins.
    expect(result.current[0]).toBe(true);

    // setState in controlled mode reports through onChange but does not
    // mutate the value locally (the parent owns it via `prop`).
    act(() => {
      result.current[1](false);
    });
    expect(onChange).toHaveBeenCalledWith(false);
    expect(result.current[0]).toBe(true);

    // Value only changes when the controlled prop changes.
    rerender({ prop: false });
    expect(result.current[0]).toBe(false);
  });

  it("seeds from defaultProp when uncontrolled and updates internal state + fires onChange", () => {
    const onChange = vi.fn();
    const { result } = renderHook(() =>
      useControllableState<boolean>({ prop: undefined, defaultProp: false, onChange }),
    );

    expect(result.current[0]).toBe(false);

    act(() => {
      result.current[1](true);
    });

    expect(result.current[0]).toBe(true);
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith(true);
  });

  it("supports updater functions when uncontrolled", () => {
    const onChange = vi.fn();
    const { result } = renderHook(() =>
      useControllableState<number>({ prop: undefined, defaultProp: 1, onChange }),
    );

    act(() => {
      result.current[1]((prev) => prev + 1);
    });

    expect(result.current[0]).toBe(2);
    expect(onChange).toHaveBeenCalledWith(2);
  });

  it("does not fire onChange when the value does not change", () => {
    const onChange = vi.fn();
    const { result } = renderHook(() =>
      useControllableState<boolean>({ prop: undefined, defaultProp: false, onChange }),
    );

    act(() => {
      result.current[1](false);
    });

    expect(onChange).not.toHaveBeenCalled();
  });

  it("switches from uncontrolled to controlled: prop takes over", () => {
    const onChange = vi.fn();
    const { result, rerender } = renderHook(
      ({ prop }: { prop: boolean | undefined }) =>
        useControllableState({ prop, defaultProp: false, onChange }),
      { initialProps: { prop: undefined as boolean | undefined } },
    );

    // Uncontrolled: local update takes effect.
    act(() => {
      result.current[1](true);
    });
    expect(result.current[0]).toBe(true);

    // Becomes controlled: the prop now dictates the value.
    rerender({ prop: false });
    expect(result.current[0]).toBe(false);

    // While controlled, setState reports through onChange without local change.
    onChange.mockClear();
    act(() => {
      result.current[1](true);
    });
    expect(onChange).toHaveBeenCalledWith(true);
    expect(result.current[0]).toBe(false);
  });
});
