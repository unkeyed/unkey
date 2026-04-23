import { cn } from "@/lib/utils";

type LogomarkProps = {
  className?: string;
};

/**
 * Unkey wordmark's "U" glyph, adapted from the marketing favicons at
 * `marketing-site/public/favicon/{light,dark}.svg`. The two favicons
 * are identical except for fill color, so we use `currentColor` here
 * and let the enclosing `text-*` utility decide light vs. dark —
 * `text-accent-12` gives near-black in light and near-white in dark.
 *
 * The wrapping span is deliberately transparent (no background) and
 * sized `size-6` (24px) so it can sit beside a breadcrumb label.
 */
export function Logomark({ className }: LogomarkProps) {
  return (
    <span
      className={cn(
        "inline-flex size-6 shrink-0 items-center justify-center text-accent-12",
        className,
      )}
      aria-label="Unkey"
    >
      <svg
        viewBox="0 0 512 512"
        fill="currentColor"
        xmlns="http://www.w3.org/2000/svg"
        className="size-5"
      >
        <path d="M170.8 115V340.6H341.2L284.4 397H170.8C139.418 397 114 371.761 114 340.6V115H170.8Z" />
        <path d="M398 284.2L341.2 340.6V115H398V284.2Z" />
      </svg>
    </span>
  );
}
