import { useMountEffect } from "~/hooks/use-mount-effect";

const TYPING_TAGS = ["INPUT", "TEXTAREA", "SELECT"];

/**
 * `F` opens the filter popover, unless the user is typing in a field or a
 * dialog owns the screen — base-ui renders those as `role="dialog"` popups
 * outside the page's DOM subtree, so nothing else would stop the shortcut
 * firing behind them.
 */
export function useFilterShortcut(setOpen: (open: boolean) => void): void {
  useMountEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "f" || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }
      const target = event.target;
      if (
        target instanceof HTMLElement &&
        (target.isContentEditable || TYPING_TAGS.includes(target.tagName))
      ) {
        return;
      }
      if (document.querySelector('[role="dialog"], [role="alertdialog"]')) {
        return;
      }
      event.preventDefault();
      setOpen(true);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  });
}
