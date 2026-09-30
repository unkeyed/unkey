import { cn } from "cn";
import type React from "react";
import type { RegionFlag } from "./region-info";

const SIZES = {
  sm: "h-3.5 w-5",
  md: "h-5 w-7",
} as const;

const US_STRIPE = 20 / 13;

const ART: Record<RegionFlag, React.ReactNode> = {
  us: (
    <>
      <rect width="28" height="20" fill="#fff" />
      {[0, 2, 4, 6, 8, 10, 12].map((stripe) => (
        <rect key={stripe} y={stripe * US_STRIPE} width="28" height={US_STRIPE} fill="#B22234" />
      ))}
      <rect width="11.2" height={US_STRIPE * 7} fill="#3C3B6E" />
      {[1.8, 4, 6.2, 8.4].map((y, row) =>
        (row % 2 === 0 ? [1.6, 4.3, 7, 9.7] : [2.95, 5.65, 8.35]).map((x) => (
          <circle key={`${x}-${y}`} cx={x} cy={y} r="0.55" fill="#fff" />
        )),
      )}
    </>
  ),
  de: (
    <>
      <rect width="28" height="6.67" fill="#000" />
      <rect y="6.67" width="28" height="6.67" fill="#DD0000" />
      <rect y="13.33" width="28" height="6.67" fill="#FFCE00" />
    </>
  ),
  sg: (
    <>
      <rect width="28" height="10" fill="#EF3340" />
      <rect y="10" width="28" height="10" fill="#fff" />
      <circle cx="6.2" cy="5" r="3" fill="#fff" />
      <circle cx="7.4" cy="5" r="2.7" fill="#EF3340" />
      {[
        [9.6, 3.1],
        [11.3, 4.3],
        [10.7, 6.3],
        [8.5, 6.3],
        [7.9, 4.3],
      ].map(([cx, cy]) => (
        <circle key={`${cx}-${cy}`} cx={cx} cy={cy} r="0.5" fill="#fff" />
      ))}
    </>
  ),
};

export function RectFlag({
  flag,
  size = "md",
}: { flag: RegionFlag | null; size?: keyof typeof SIZES }) {
  return (
    <span className="inline-flex shrink-0 p-[2px] ring-1 ring-grayA-6 ring-inset">
      <span className={cn("inline-flex overflow-hidden", SIZES[size])}>
        {flag === null ? (
          <span className="size-full bg-grayA-3" />
        ) : (
          <svg viewBox="0 0 28 20" className="size-full" aria-hidden="true">
            {ART[flag]}
          </svg>
        )}
      </span>
    </span>
  );
}
