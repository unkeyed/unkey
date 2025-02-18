import { useEffect } from "react";

type KeyCombo = {
  key: string;
  ctrl?: boolean;
  meta?: boolean;
  shift?: boolean;
  alt?: boolean;
};

type KeyboardShortcutOptions = {
  preventDefault?: boolean;
  ignoreInputs?: boolean;
  ignoreContentEditable?: boolean;
};

const defaultOptions: KeyboardShortcutOptions = {
  preventDefault: true,
  ignoreInputs: true,
  ignoreContentEditable: true,
};

export function useKeyboardShortcut(
  shortcut: string | KeyCombo,
  callback: () => void,
  options: KeyboardShortcutOptions = defaultOptions,
) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Convert simple string shortcut to KeyCombo
      const combo = typeof shortcut === "string" ? { key: shortcut } : shortcut;

      // Normalize the key to lowercase for comparison
      const keyMatch = e.key.toLowerCase() === combo.key.toLowerCase();

      // Check if any modifier keys are pressed when they're not part of the shortcut
      const hasUnwantedModifiers =
        (combo.ctrl === undefined && e.ctrlKey) ||
        (combo.meta === undefined && e.metaKey) ||
        (combo.shift === undefined && e.shiftKey) ||
        (combo.alt === undefined && e.altKey);

      // If unwanted modifiers are pressed, don't trigger the shortcut
      if (hasUnwantedModifiers) {
        return;
      }

      // Check modifier keys if specified
      const ctrlMatch = combo.ctrl === undefined || e.ctrlKey === combo.ctrl;
      const metaMatch = combo.meta === undefined || e.metaKey === combo.meta;
      const shiftMatch = combo.shift === undefined || e.shiftKey === combo.shift;
      const altMatch = combo.alt === undefined || e.altKey === combo.alt;

      // Check if we should ignore based on target
      if (
        options.ignoreInputs &&
        (e.target instanceof HTMLInputElement ||
          e.target instanceof HTMLTextAreaElement ||
          e.target instanceof HTMLSelectElement)
      ) {
        return;
      }

      // Check for contentEditable if option is set
      if (options.ignoreContentEditable && (e.target as HTMLElement).isContentEditable) {
        return;
      }

      // If all conditions match, execute callback
      if (keyMatch && ctrlMatch && metaMatch && shiftMatch && altMatch) {
        if (options.preventDefault) {
          e.preventDefault();
        }
        callback();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [shortcut, callback, options]);
}
