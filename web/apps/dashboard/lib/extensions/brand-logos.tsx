/**
 * Inline brand marks for extensions whose brands either are not on
 * simpleicons (Axiom, Slack are excluded for trademark reasons) or render
 * poorly as flat single-color CDN SVGs.
 *
 * Marks use `currentColor` so they pick up the wrapper text color, and
 * each entry sets that color via a className on the wrapper.
 */
"use client";

import { cn } from "@unkey/ui/src/lib/utils";

type BrandMark = {
  /** Tailwind text color class applied to the wrapper. */
  colorClass: string;
  Logo: React.ComponentType<{ className?: string }>;
};

function AxiomMark({ className }: { className?: string }) {
  // Official Axiom mark from axiom.co footer. Uses currentColor for theming.
  return (
    <svg className={className} viewBox="0 0 17 15" fill="currentColor" aria-hidden>
      <path d="M16.5089 10.1066L13.0911 4.31803C12.9344 4.05199 12.5482 3.83432 12.2329 3.83432H10.0991C9.60314 3.83432 9.39981 3.49237 9.64721 3.07442L10.8173 1.0978C10.9102 0.940926 10.91 0.747804 10.8168 0.5911C10.7236 0.434397 10.5516 0.337891 10.3655 0.337891H7.38875C7.07344 0.337891 6.68637 0.555072 6.52858 0.820524L0.744369 10.5524C0.586609 10.8178 0.586487 11.2522 0.744156 11.5177L2.23248 14.0243C2.48046 14.442 2.88713 14.4425 3.13616 14.0254L4.29915 12.0781C4.54819 11.661 4.95486 11.6615 5.20283 12.0792L6.25715 13.8548C6.41479 14.1203 6.80177 14.3376 7.11707 14.3376H13.9955C14.3109 14.3376 14.6978 14.1203 14.8555 13.8548L16.5072 11.0731C16.6649 10.8075 16.6656 10.3726 16.5089 10.1066ZM11.8932 9.828C12.1396 10.2465 11.9355 10.5889 11.4395 10.5889H6.08915C5.5932 10.5889 5.39029 10.2472 5.63826 9.82956L8.31555 5.32067C8.56352 4.90304 8.96929 4.90305 9.21723 5.3207L11.8932 9.828Z" />
    </svg>
  );
}

function SlackMark({ className }: { className?: string }) {
  // Multi-color Slack mark.
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zM6.313 15.165a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313z"
        fill="#E01E5A"
      />
      <path
        d="M8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zM8.834 6.313a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312z"
        fill="#36C5F0"
      />
      <path
        d="M18.956 8.834a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zM17.688 8.834a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.165 0a2.528 2.528 0 0 1 2.523 2.522v6.312z"
        fill="#2EB67D"
      />
      <path
        d="M15.165 18.956a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.165 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zM15.165 17.688a2.527 2.527 0 0 1-2.52-2.523 2.526 2.526 0 0 1 2.52-2.52h6.313A2.527 2.527 0 0 1 24 15.165a2.528 2.528 0 0 1-2.522 2.523h-6.313z"
        fill="#ECB22E"
      />
    </svg>
  );
}

const BRAND_MARKS: Record<string, BrandMark> = {
  axiom: { colorClass: "text-[#1A8FE4]", Logo: AxiomMark },
  slack: { colorClass: "", Logo: SlackMark },
};

type BrandLogoProps = {
  /** Extension slug; if a custom mark is registered above, it wins over `iconUrl`. */
  slug: string;
  /** Fallback URL (e.g. simpleicons CDN) when no inline mark exists. */
  iconUrl?: string;
  /** Display name used to render initials when no logo is available. */
  name: string;
  /** Tailwind size class for the rendered mark. */
  className?: string;
};

export function BrandLogo({ slug, iconUrl, name, className }: BrandLogoProps) {
  const mark = BRAND_MARKS[slug];
  if (mark) {
    const Logo = mark.Logo;
    return (
      <span className={cn("inline-flex items-center justify-center", mark.colorClass)}>
        <Logo className={className} />
      </span>
    );
  }
  if (iconUrl) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img src={iconUrl} alt="" className={cn("object-contain", className)} loading="lazy" />
    );
  }
  return (
    <span
      className={cn(
        "inline-flex items-center justify-center rounded-md bg-grayA-3 text-[10px] font-mono font-medium text-accent-12",
        className,
      )}
    >
      {name.slice(0, 2).toUpperCase()}
    </span>
  );
}
