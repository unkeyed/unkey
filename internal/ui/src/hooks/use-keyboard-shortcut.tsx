import { useCallback, useEffect } from "react";

/**
 * Represents the parsed details of a keyboard shortcut.
 */
type KeyCombo = {
  key: string; // Original key name (e.g., 'e', 'enter') - useful for display
  code?: string | null; // Expected KeyboardEvent.code (e.g., 'KeyE', 'Enter') - used for matching
  ctrl: boolean; // True if Ctrl key is required
  meta: boolean; // True if Meta key (Cmd/Win) is required
  shift: boolean; // True if Shift key is required
  alt: boolean; // True if Alt key (Option on Mac) is required
};

/**
 * Configuration options for the useKeyboardShortcut hook.
 */
type KeyboardShortcutOptions = {
  preventDefault?: boolean; // Call event.preventDefault() if shortcut matches (default: true)
  ignoreInputs?: boolean; // Ignore shortcuts if focus is on input/textarea/select (default: true)
  ignoreContentEditable?: boolean; // Ignore shortcuts if focus is on contentEditable element (default: true)
  disabled?: boolean; // Disable the shortcut listener entirely (default: false)
};

// Default options for the hook
const defaultOptions: Required<KeyboardShortcutOptions> = {
  preventDefault: true,
  ignoreInputs: true,
  ignoreContentEditable: true,
  disabled: false,
};

/**
 * Maps common key names (like 'a', 'enter', 'f1', 'comma') to their
 * corresponding KeyboardEvent.code values (like 'KeyA', 'Enter', 'F1', 'Comma').
 * Expand this function to support more keys as needed.
 * @param keyName The user-friendly key name from the shortcut string.
 * @returns The corresponding KeyboardEvent.code string, or null if not found.
 */
export const getKeyNameToCode = (keyName: string): string | null => {
  const lowerKey = keyName.toLowerCase();

  // Basic Letters (A-Z)
  if (lowerKey.length === 1 && lowerKey >= "a" && lowerKey <= "z") {
    return `Key${lowerKey.toUpperCase()}`; // 'a' -> 'KeyA'
  }

  // Digits (0-9) - Main number row
  if (lowerKey.length === 1 && lowerKey >= "0" && lowerKey <= "9") {
    return `Digit${lowerKey}`; // '1' -> 'Digit1'
  }

  // Common Control/Whitespace Keys
  switch (lowerKey) {
    case "enter":
      return "Enter";
    case "tab":
      return "Tab";
    case "space":
      return "Space"; // Use 'space' keyword
    case "esc":
    case "escape":
      return "Escape";
    case "backspace":
      return "Backspace";
    case "delete":
      return "Delete";
    // Add more common keys as needed (e.g., Home, End, PageUp, PageDown)
  }

  // Arrow Keys
  switch (lowerKey) {
    case "up":
    case "arrowup":
      return "ArrowUp";
    case "down":
    case "arrowdown":
      return "ArrowDown";
    case "left":
    case "arrowleft":
      return "ArrowLeft";
    case "right":
    case "arrowright":
      return "ArrowRight";
  }

  if (lowerKey.startsWith("f") && !Number.isNaN(Number.parseInt(lowerKey.substring(1), 10))) {
    const fNum = Number.parseInt(lowerKey.substring(1), 10);
    if (fNum >= 1 && fNum <= 12) {
      return `F${fNum}`;
    }
  }

  switch (lowerKey) {
    case ",":
    case "comma":
      return "Comma";
    case ".":
    case "period":
      return "Period";
    case "/":
    case "slash":
      return "Slash";
    case ";":
    case "semicolon":
      return "Semicolon";
    case "'":
    case "quote":
      return "Quote";
    case "[":
    case "bracketleft":
      return "BracketLeft";
    case "]":
    case "bracketright":
      return "BracketRight";
    case "\\":
    case "backslash":
      return "Backslash";
    case "`":
    case "backquote":
      return "Backquote";
    case "-":
    case "minus":
      return "Minus";
    case "=":
    case "equal":
      return "Equal";
  }

  console.warn(
    `[useKeyboardShortcut] Could not map key name "${keyName}" to a standard KeyboardEvent.code. You might need to use the code directly in the shortcut definition or expand the mapping.`,
  );
  return null; // Indicate failure to map
};

/**
 * Parses a shortcut string (e.g., "ctrl+shift+k") into a KeyCombo object.
 * It now maps the key name to its KeyboardEvent.code.
 * @param shortcut The shortcut string to parse.
 * @returns A KeyCombo object or null if parsing fails.
 */
export const parseShortcutString = (shortcut: string): KeyCombo | null => {
  if (!shortcut || typeof shortcut !== "string") {
    return null;
  }

  const parts = shortcut
    .toLowerCase()
    .split("+")
    .map((part) => part.trim());

  const combo: Partial<KeyCombo> & { key?: string; code?: string | null } = {};
  let keyAssigned = false;

  for (const part of parts) {
    switch (part) {
      case "ctrl":
      case "control":
        combo.ctrl = true;
        break;
      case "shift":
        combo.shift = true;
        break;
      case "meta":
      case "cmd":
      case "win":
        combo.meta = true;
        break;
      case "alt":
      case "option":
        combo.alt = true;
        break;
      default:
        if (part.length > 0) {
          if (keyAssigned) {
            console.warn(
              `[useKeyboardShortcut] Multiple non-modifier keys detected in shortcut: "${shortcut}"`,
            );
            return null;
          }
          combo.key = part; // Store original key name
          combo.code = getKeyNameToCode(part); // Attempt to get the code
          if (!combo.code) {
            console.warn(
              `[useKeyboardShortcut] Failed to map key "${part}" to code for shortcut: "${shortcut}". Please check spelling or expand getKeyNameToCode mapping.`,
            );
            return null; // Fail parsing if code cannot be determined
          }
          keyAssigned = true;
        } else {
          console.warn(
            `[useKeyboardShortcut] Empty part detected in shortcut string: "${shortcut}"`,
          );
          return null;
        }
    }
  }

  // Final validation: Ensure a key and code were actually assigned
  if (!keyAssigned || !combo.key || !combo.code) {
    console.warn(`[useKeyboardShortcut] No valid key/code identified for shortcut: "${shortcut}"`);
    return null;
  }

  // Return the full KeyCombo object with explicit boolean values for modifiers
  return {
    key: combo.key,
    code: combo.code,
    ctrl: !!combo.ctrl,
    meta: !!combo.meta,
    shift: !!combo.shift,
    alt: !!combo.alt,
  };
};

/**
 * A React hook to trigger a callback function when a specific keyboard shortcut is pressed globally.
 * Matches shortcuts based on KeyboardEvent.code (physical key) for better reliability, especially with Alt/Option keys.
 * Handles complex shortcut strings (e.g., "ctrl+shift+k") or pre-defined KeyCombo objects.
 * Includes options to ignore inputs, contentEditable elements, prevent default behavior, and disable the listener.
 *
 * @param shortcut The keyboard shortcut definition (string like "ctrl+alt+e", "shift+enter", or KeyCombo object). Can be null/undefined to disable. Key names should be standard (e.g., 'a', 'enter', 'f1', 'comma').
 * @param callback The function to execute when the shortcut is matched. Should be memoized (e.g., with useCallback) if defined inline or stable otherwise.
 * @param options Optional configuration for the shortcut listener (KeyboardShortcutOptions). Options are merged with defaults. Provide a stable reference (e.g., useMemo) if passing an object.
 */
export function useKeyboardShortcut(
  shortcut: string | KeyCombo | null | undefined,
  callback: () => void,
  options?: KeyboardShortcutOptions,
): void {
  const mergedOptions = { ...defaultOptions, ...options };

  const { preventDefault, ignoreInputs, ignoreContentEditable, disabled } = mergedOptions;

  // Memoize callback for stability in the useEffect dependency array
  // biome-ignore lint/correctness/useExhaustiveDependencies: <explanation>
  const memoizedCallback = useCallback(callback, [callback]);

  useEffect(() => {
    // Parse the shortcut definition inside the effect.
    // This ensures we work with a stable KeyCombo object within this effect run.
    let parsedCombo: KeyCombo | null = null;
    if (typeof shortcut === "string") {
      parsedCombo = parseShortcutString(shortcut);
    } else if (shortcut?.key && shortcut.code) {
      // If a KeyCombo object is passed, ensure modifier flags are boolean
      parsedCombo = {
        ...shortcut,
        ctrl: !!shortcut.ctrl,
        meta: !!shortcut.meta,
        shift: !!shortcut.shift,
        alt: !!shortcut.alt,
      };
    } else if (shortcut) {
      // Handle potentially invalid KeyCombo objects passed directly
      console.warn("[useKeyboardShortcut] Invalid KeyCombo object provided:", shortcut);
    }

    // Exit if the hook is disabled or the shortcut definition is invalid/unparsed
    if (disabled || !parsedCombo) {
      return; // No listener will be attached
    }

    const requiredCombo = parsedCombo;

    const handleKeyDown = (e: KeyboardEvent): void => {
      const target = e.target as HTMLElement;
      if (
        ignoreInputs &&
        (target instanceof HTMLInputElement ||
          target instanceof HTMLTextAreaElement ||
          target instanceof HTMLSelectElement)
      ) {
        return; // Ignore if focus is on standard input elements
      }
      if (ignoreContentEditable && target.isContentEditable) {
        return; // Ignore if focus is on a contentEditable element
      }

      // Match the physical key code (more reliable than e.key with modifiers)
      const codeMatch = e.code === requiredCombo.code;

      // Match modifier keys state
      const ctrlMatch = e.ctrlKey === requiredCombo.ctrl;
      const metaMatch = e.metaKey === requiredCombo.meta; // Cmd on Mac, Win key on Windows
      const shiftMatch = e.shiftKey === requiredCombo.shift;
      const altMatch = e.altKey === requiredCombo.alt; // Option on Mac

      // Check if all conditions are met
      if (codeMatch && ctrlMatch && metaMatch && shiftMatch && altMatch) {
        if (preventDefault) {
          e.preventDefault(); // Prevent default browser action if configured
        }
        memoizedCallback();
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [shortcut, memoizedCallback, preventDefault, ignoreInputs, ignoreContentEditable, disabled]);
}
