import { useState } from "react";

interface Props {
  /**
   * Name of the scale as it appears in the CSS variables and Tailwind
   * utilities — `gray`, `accent`, `success`, etc. Must correspond to
   * variables defined on `:root` and `.dark`.
   */
  scale: string;
  /**
   * Set to `true` for scales defined as `hsla(var(--foo))` (alpha scales —
   * `grayA`, `blackA`, `whiteA`). The rendered swatch uses `hsla()` in that
   * case so the alpha channel in the variable is honored.
   */
  alpha?: boolean;
}

const STEPS = Array.from({ length: 12 }, (_, i) => i + 1);

/**
 * ColorSwatch renders the twelve steps of a Radix-style color scale as a
 * bordered grid, with each swatch clickable to copy its token name to the
 * clipboard.
 *
 * Why it takes `scale` as a string rather than 12 concrete colors: the
 * swatches read the CSS variable at runtime via inline `style={{ background
 * }}`, so the grid stays in sync with whatever the UI package exports —
 * including dark-mode values, alpha variants, and any future scale added
 * without touching this file.
 *
 * Step numbers are rendered below each swatch (not overlaid) so they stay
 * readable regardless of background luminance; `mix-blend-difference` on
 * mid-range colors produced unreadable grays.
 */
export function ColorSwatch({ scale, alpha = false }: Props) {
  return (
    <div className="mt-10 mb-4 grid grid-cols-6 md:grid-cols-12 gap-3 rounded-lg bg-background border border-grayA-3 pt-10 pb-14 px-8">
      {STEPS.map((step) => (
        <SwatchCell key={step} scale={scale} alpha={alpha} step={step} />
      ))}
    </div>
  );
}

/**
 * Single swatch cell. Renders a 1:3 aspect rectangle tinted via inline
 * `style`, with a hover tag showing the token name and a click-to-copy
 * handler.
 *
 * Kept private because the token string and alpha/non-alpha split are
 * implementation details of the grid — external callers work with whole
 * scales, not individual steps.
 */
function SwatchCell({
  scale,
  alpha,
  step,
}: {
  scale: string;
  alpha: boolean;
  step: number;
}) {
  const token = `${scale}-${step}`;
  const cssFn = alpha ? "hsla" : "hsl";
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(token);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch {
      // `writeText` can reject when the page isn't served over a secure
      // context (http:// outside localhost) or the document lacks focus.
      // Failing silently is the right UX — the hover tag still shows the
      // token so the user can copy it manually.
    }
  }

  return (
    <button
      type="button"
      onClick={copy}
      aria-label={token}
      className="group relative aspect-[1/3] rounded-md cursor-pointer"
      style={{ background: `${cssFn}(var(--${token}))` }}
    >
      <span className="absolute left-1/2 -translate-x-1/2 -bottom-6 font-mono text-[10px] text-gray-10 pointer-events-none">
        {step}
      </span>
      <span
        className={`absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-2 py-1 rounded-[4px] bg-gray-12 text-gray-1 font-mono text-[10px] whitespace-nowrap transition-opacity duration-150 pointer-events-none z-10 ${
          copied ? "opacity-0" : "opacity-0 group-hover:opacity-100"
        }`}
      >
        {token}
      </span>
      <span
        className={`absolute inset-0 flex items-center justify-center rounded-md bg-gray-12 text-gray-1 text-[10px] font-medium uppercase tracking-[0.24em] transition-opacity pointer-events-none ${
          copied ? "opacity-100" : "opacity-0"
        }`}
      >
        Copied
      </span>
    </button>
  );
}
