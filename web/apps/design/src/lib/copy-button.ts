const BUTTON_CLASS =
  "inline-flex h-7 items-center rounded-md px-2 font-sans text-xs font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors";
const FEEDBACK_MS = 1500;

interface Options {
  /** Tailwind top offset class. Defaults to `top-3`. Use a larger value
   *  (`top-11`) when the container's top is overlapped by another layer. */
  topClass?: string;
  /** Place the button in normal flow instead of absolutely positioned.
   *  The container is left untouched (no `position: relative`). */
  inline?: boolean;
}

/**
 * Append a "Copy" button into `container`. Container is given
 * `position: relative` (if not already positioned) so the button sits
 * at top-right. The button copies `text` to the clipboard and briefly
 * flashes "Copied" on click.
 *
 * Returns a cleanup function that removes the button — useful for React
 * effects that re-run on prop changes.
 */
export function attachCopyButton(
  container: HTMLElement,
  text: string,
  options: Options = {},
): () => void {
  if (container.dataset.copyEnhanced) {
    return () => {};
  }
  container.dataset.copyEnhanced = "true";

  const previousPosition = container.style.position;
  if (!options.inline && !previousPosition) {
    container.style.position = "relative";
  }

  const btn = document.createElement("button");
  btn.type = "button";
  btn.className = options.inline
    ? BUTTON_CLASS
    : `${BUTTON_CLASS} absolute right-3 ${options.topClass ?? "top-3"}`;
  btn.textContent = "Copy";
  btn.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(text);
      btn.textContent = "Copied";
      setTimeout(() => {
        btn.textContent = "Copy";
      }, FEEDBACK_MS);
    } catch (_err) {}
  });
  container.appendChild(btn);

  return () => {
    btn.remove();
    container.style.position = previousPosition;
    delete container.dataset.copyEnhanced;
  };
}
