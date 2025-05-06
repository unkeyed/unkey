import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { parseShortcutString, useKeyboardShortcut } from "./use-keyboard-shortcut";

function dispatchKeyEvent(
  key: string,
  code: string,
  options: {
    ctrlKey?: boolean;
    metaKey?: boolean;
    shiftKey?: boolean;
    altKey?: boolean;
    target?: Element;
  } = {},
) {
  const event = new KeyboardEvent("keydown", {
    key: key,
    code: code,
    ctrlKey: options.ctrlKey ?? false,
    metaKey: options.metaKey ?? false,
    shiftKey: options.shiftKey ?? false,
    altKey: options.altKey ?? false,
    bubbles: true,
    cancelable: true,
  });

  vi.spyOn(event, "preventDefault");

  act(() => {
    (options.target ?? document).dispatchEvent(event);
  });

  return event;
}

describe("Keyboard Shortcut Functionality", () => {
  describe("useKeyboardShortcut Hook", () => {
    let callback: ReturnType<typeof vi.fn>;

    beforeEach(() => {
      callback = vi.fn();
      vi.restoreAllMocks();
    });

    it("should call callback when the correct key is pressed", () => {
      renderHook(() => useKeyboardShortcut("k", callback));
      dispatchKeyEvent("k", "KeyK");
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should call callback when the correct key combination (Ctrl+S) is pressed", () => {
      renderHook(() => useKeyboardShortcut("ctrl+s", callback));
      dispatchKeyEvent("s", "KeyS", { ctrlKey: true });
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should call callback when the correct key combination (Shift+Alt+E) is pressed", () => {
      renderHook(() => useKeyboardShortcut("shift+alt+e", callback));
      dispatchKeyEvent("e", "KeyE", { shiftKey: true, altKey: true });
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should call callback when the correct key combination (Cmd+K) is pressed", () => {
      renderHook(() => useKeyboardShortcut("meta+k", callback));
      dispatchKeyEvent("k", "KeyK", { metaKey: true });
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should not call callback when the wrong key is pressed", () => {
      renderHook(() => useKeyboardShortcut("k", callback));
      dispatchKeyEvent("j", "KeyJ");
      expect(callback).not.toHaveBeenCalled();
    });

    it("should not call callback when modifiers are missing", () => {
      renderHook(() => useKeyboardShortcut("ctrl+k", callback));
      dispatchKeyEvent("k", "KeyK");
      expect(callback).not.toHaveBeenCalled();
    });

    it("should not call callback when extra modifiers are pressed", () => {
      renderHook(() => useKeyboardShortcut("k", callback));
      dispatchKeyEvent("k", "KeyK", { ctrlKey: true });
      expect(callback).not.toHaveBeenCalled();
    });

    it("should handle special keys like Enter", () => {
      renderHook(() => useKeyboardShortcut("enter", callback));
      dispatchKeyEvent("Enter", "Enter");
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should handle special keys like Comma", () => {
      renderHook(() => useKeyboardShortcut(",", callback));
      dispatchKeyEvent(",", "Comma");
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should handle function keys like F5", () => {
      renderHook(() => useKeyboardShortcut("f5", callback));
      dispatchKeyEvent("F5", "F5");
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should properly reject modifier-only shortcuts", () => {
      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      renderHook(() => useKeyboardShortcut("ctrl+shift", callback));
      dispatchKeyEvent("Shift", "ShiftLeft", { ctrlKey: true, shiftKey: true });
      expect(callback).not.toHaveBeenCalled();
      consoleSpy.mockRestore();
    });

    it("should call event.preventDefault() by default", () => {
      renderHook(() => useKeyboardShortcut("p", callback));
      const event = dispatchKeyEvent("p", "KeyP");
      expect(callback).toHaveBeenCalledTimes(1);
      expect(event.preventDefault).toHaveBeenCalledTimes(1);
    });

    describe("ignoreInputs option", () => {
      let input: HTMLInputElement;
      let textarea: HTMLTextAreaElement;
      let select: HTMLSelectElement;

      beforeEach(() => {
        input = document.createElement("input");
        textarea = document.createElement("textarea");
        select = document.createElement("select");
        document.body.appendChild(input);
        document.body.appendChild(textarea);
        document.body.appendChild(select);
        input.focus();
      });

      afterEach(() => {
        document.body.removeChild(input);
        document.body.removeChild(textarea);
        document.body.removeChild(select);
      });

      it("should ignore shortcut when focus is on an input element by default", () => {
        renderHook(() => useKeyboardShortcut("i", callback));
        dispatchKeyEvent("i", "KeyI", { target: input });
        expect(callback).not.toHaveBeenCalled();
      });

      it("should ignore shortcut when focus is on a textarea element by default", () => {
        renderHook(() => useKeyboardShortcut("t", callback));
        textarea.focus();
        dispatchKeyEvent("t", "KeyT", { target: textarea });
        expect(callback).not.toHaveBeenCalled();
      });

      it("should ignore shortcut when focus is on a select element by default", () => {
        renderHook(() => useKeyboardShortcut("s", callback));
        select.focus();
        dispatchKeyEvent("s", "KeyS", { target: select });
        expect(callback).not.toHaveBeenCalled();
      });

      it("should NOT ignore shortcut from input when ignoreInputs is false", () => {
        renderHook(() => useKeyboardShortcut("i", callback, { ignoreInputs: false }));
        dispatchKeyEvent("i", "KeyI", { target: input });
        expect(callback).toHaveBeenCalledTimes(1);
      });

      it("should NOT ignore shortcut from textarea when ignoreInputs is false", () => {
        renderHook(() => useKeyboardShortcut("t", callback, { ignoreInputs: false }));
        textarea.focus();
        dispatchKeyEvent("t", "KeyT", { target: textarea });
        expect(callback).toHaveBeenCalledTimes(1);
      });

      it("should NOT ignore shortcut from select when ignoreInputs is false", () => {
        renderHook(() => useKeyboardShortcut("s", callback, { ignoreInputs: false }));
        select.focus();
        dispatchKeyEvent("s", "KeyS", { target: select });
        expect(callback).toHaveBeenCalledTimes(1);
      });
    });

    describe("ignoreContentEditable option", () => {
      let editableDiv: HTMLDivElement;

      beforeEach(() => {
        editableDiv = document.createElement("div");
        editableDiv.setAttribute("contenteditable", "true");
        Object.defineProperty(editableDiv, "isContentEditable", {
          value: true,
          writable: false,
        });
        document.body.appendChild(editableDiv);
        editableDiv.focus();
      });

      afterEach(() => {
        document.body.removeChild(editableDiv);
      });

      it("should ignore shortcut when focus is on a contentEditable element by default", () => {
        renderHook(() => useKeyboardShortcut("c", callback));
        dispatchKeyEvent("c", "KeyC", { target: editableDiv });
        expect(callback).not.toHaveBeenCalled();
      });

      it("should NOT ignore shortcut from contentEditable when ignoreContentEditable is false", () => {
        renderHook(() => useKeyboardShortcut("c", callback, { ignoreContentEditable: false }));
        dispatchKeyEvent("c", "KeyC", { target: editableDiv });
        expect(callback).toHaveBeenCalledTimes(1);
      });
    });

    it("should not attach listener or call callback if disabled is true", () => {
      const addSpy = vi.spyOn(document, "addEventListener");
      renderHook(() => useKeyboardShortcut("d", callback, { disabled: true }));
      dispatchKeyEvent("d", "KeyD");
      expect(addSpy).not.toHaveBeenCalledWith("keydown", expect.any(Function));
      expect(callback).not.toHaveBeenCalled();
      addSpy.mockRestore();
    });

    it("should not call callback if shortcut is null or undefined", () => {
      renderHook(() => useKeyboardShortcut(null, callback));
      dispatchKeyEvent("a", "KeyA");
      expect(callback).not.toHaveBeenCalled();

      renderHook(() => useKeyboardShortcut(undefined, callback));
      dispatchKeyEvent("b", "KeyB");
      expect(callback).not.toHaveBeenCalled();
    });

    it("should clean up event listener on unmount", () => {
      const addSpy = vi.spyOn(document, "addEventListener");
      const removeSpy = vi.spyOn(document, "removeEventListener");

      const { unmount } = renderHook(() => useKeyboardShortcut("u", callback));

      const listener = addSpy.mock.calls.find((call) => call[0] === "keydown")?.[1];
      expect(listener).toBeDefined();
      expect(addSpy).toHaveBeenCalledWith("keydown", listener);

      unmount();

      expect(removeSpy).toHaveBeenCalledWith("keydown", listener);

      addSpy.mockRestore();
      removeSpy.mockRestore();
    });

    it("should update listener if shortcut changes", () => {
      const addSpy = vi.spyOn(document, "addEventListener");
      const removeSpy = vi.spyOn(document, "removeEventListener");
      let shortcut = "a";

      const { rerender } = renderHook(() => useKeyboardShortcut(shortcut, callback));
      const listener1 = addSpy.mock.calls.find((call) => call[0] === "keydown")?.[1];

      dispatchKeyEvent("a", "KeyA");
      expect(callback).toHaveBeenCalledTimes(1);
      dispatchKeyEvent("b", "KeyB");
      expect(callback).toHaveBeenCalledTimes(1);

      shortcut = "b";
      rerender();

      expect(removeSpy).toHaveBeenCalledWith("keydown", listener1);
      const listener2 = addSpy.mock.calls.slice(-1)[0][1];
      expect(addSpy).toHaveBeenCalledWith("keydown", listener2);
      expect(listener1).not.toBe(listener2);

      dispatchKeyEvent("a", "KeyA");
      expect(callback).toHaveBeenCalledTimes(1);
      dispatchKeyEvent("b", "KeyB");
      expect(callback).toHaveBeenCalledTimes(2);

      addSpy.mockRestore();
      removeSpy.mockRestore();
    });

    it("should accept a KeyCombo object as input", () => {
      const keyCombo = {
        key: "x",
        code: "KeyX",
        ctrl: true,
        meta: false,
        shift: true,
        alt: false,
      };
      renderHook(() => useKeyboardShortcut(keyCombo, callback));

      dispatchKeyEvent("x", "KeyX", { ctrlKey: true, shiftKey: true });
      expect(callback).toHaveBeenCalledTimes(1);

      dispatchKeyEvent("x", "KeyX", { ctrlKey: true });
      expect(callback).toHaveBeenCalledTimes(1);
    });
  });

  describe("parseShortcutString Function", () => {
    let warnSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
      //@ts-expect-error should be fine
      warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    });

    afterEach(() => {
      warnSpy.mockRestore();
    });

    it("should parse a single key correctly (lowercase)", () => {
      expect(parseShortcutString("k")).toEqual({
        key: "k",
        code: "KeyK",
        ctrl: false,
        meta: false,
        shift: false,
        alt: false,
      });
    });

    it("should parse a single key correctly (uppercase)", () => {
      expect(parseShortcutString("K")).toEqual({
        key: "k",
        code: "KeyK",
        ctrl: false,
        meta: false,
        shift: false,
        alt: false,
      });
    });

    it("should parse combined modifiers and key", () => {
      expect(parseShortcutString("ctrl+shift+alt+meta+f")).toEqual({
        key: "f",
        code: "KeyF",
        ctrl: true,
        meta: true,
        shift: true,
        alt: true,
      });
    });

    it("should parse number keys", () => {
      expect(parseShortcutString("1")).toEqual({
        key: "1",
        code: "Digit1",
        ctrl: false,
        meta: false,
        shift: false,
        alt: false,
      });
    });

    it("should parse special mapped keys (e.g., space, escape)", () => {
      expect(parseShortcutString("space")).toEqual({
        key: "space",
        code: "Space",
        ctrl: false,
        meta: false,
        shift: false,
        alt: false,
      });
      expect(parseShortcutString("shift+escape")).toEqual({
        key: "escape",
        code: "Escape",
        ctrl: false,
        meta: false,
        shift: true,
        alt: false,
      });
    });

    it("should parse punctuation keys (e.g., comma, bracketleft)", () => {
      expect(parseShortcutString(",")).toEqual({
        key: ",",
        code: "Comma",
        ctrl: false,
        meta: false,
        shift: false,
        alt: false,
      });
      expect(parseShortcutString("ctrl+BracketLeft")).toEqual({
        key: "bracketleft",
        code: "BracketLeft",
        ctrl: true,
        meta: false,
        shift: false,
        alt: false,
      });
    });

    it("should handle modifier aliases (cmd, option, control, win)", () => {
      expect(parseShortcutString("cmd+k")).toEqual({
        key: "k",
        code: "KeyK",
        ctrl: false,
        meta: true,
        shift: false,
        alt: false,
      });
      expect(parseShortcutString("option+k")).toEqual({
        key: "k",
        code: "KeyK",
        ctrl: false,
        meta: false,
        shift: false,
        alt: true,
      });
      expect(parseShortcutString("control+k")).toEqual({
        key: "k",
        code: "KeyK",
        ctrl: true,
        meta: false,
        shift: false,
        alt: false,
      });
      expect(parseShortcutString("win+k")).toEqual({
        key: "k",
        code: "KeyK",
        ctrl: false,
        meta: true,
        shift: false,
        alt: false,
      });
    });

    it("should handle whitespace and mixed case", () => {
      expect(parseShortcutString(" Ctrl + SHIFT + a ")).toEqual({
        key: "a",
        code: "KeyA",
        ctrl: true,
        meta: false,
        shift: true,
        alt: false,
      });
    });

    it("should return null for null or undefined input", () => {
      expect(parseShortcutString(null as any)).toBeNull();
      expect(parseShortcutString(undefined as any)).toBeNull();
      expect(warnSpy).not.toHaveBeenCalled();
    });

    it("should return null for empty string input", () => {
      expect(parseShortcutString("")).toBeNull();
      expect(warnSpy).not.toHaveBeenCalled();
    });

    it("should return null for non-string input", () => {
      expect(parseShortcutString(123 as any)).toBeNull();
      expect(parseShortcutString({} as any)).toBeNull();
      expect(warnSpy).not.toHaveBeenCalled();
    });

    it("should return null and warn for empty parts", () => {
      expect(parseShortcutString("ctrl++k")).toBeNull();
      expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining("Empty part detected"));
    });

    it("should return null and warn for multiple non-modifier keys", () => {
      expect(parseShortcutString("a+b")).toBeNull();
      expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining("Multiple non-modifier keys"));
      warnSpy.mockClear();
      expect(parseShortcutString("ctrl+a+b")).toBeNull();
      expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining("Multiple non-modifier keys"));
    });

    it("should return null and warn if only modifiers are provided", () => {
      expect(parseShortcutString("ctrl+shift+")).toBeNull();
      expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining("Empty part detected"));
      warnSpy.mockClear();
      expect(parseShortcutString("ctrl+shift")).toBeNull();
      expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining("No valid key/code identified"));
    });

    it("should return null and warn for unmappable key names", () => {
      expect(parseShortcutString("ctrl+unknownkey")).toBeNull();
      expect(warnSpy).toHaveBeenCalledWith(
        expect.stringContaining('Failed to map key "unknownkey"'),
      );
    });

    it("should return null and warn for unknown modifiers treated as keys", () => {
      expect(parseShortcutString("super+k")).toBeNull();
      expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('Failed to map key "super"'));
      warnSpy.mockClear();

      expect(parseShortcutString("k+super")).toBeNull();
      expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining("Multiple non-modifier keys"));
    });
  });
});
