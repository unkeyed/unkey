import type { FlagCode } from "@/lib/trpc/routers/deploy/network/utils";
import { cn } from "@/lib/utils";
import { Laptop2 } from "@unkey/icons";

type RegionFlagSize = "xs" | "sm" | "md" | "lg";
type RegionFlagShape = "rounded" | "circle" | "square";
type RegionFlagProps = {
  flagCode: FlagCode;
  size?: RegionFlagSize;
  shape?: RegionFlagShape;
  className?: string;
};

const sizeClass: Record<RegionFlagSize, string> = {
  xs: "size-4",
  sm: "size-[22px]",
  md: "size-9",
  lg: "size-12",
};

// Height for the rectangular ("square") variant; width follows the flag's
// natural 4:3 aspect.
const rectHeight: Record<RegionFlagSize, string> = {
  xs: "h-3.5",
  sm: "h-[18px]",
  md: "h-7",
  lg: "h-9",
};

const shapeClass: Record<Exclude<RegionFlagShape, "square">, string> = {
  rounded: "rounded-[10px]",
  circle: "rounded-full",
};

// "local" has no country flag; like the "Global → globe" convention, it gets a
// neutral glyph (a laptop = your own machine) in the same slot.
const localIconSize = {
  xs: "sm-regular",
  sm: "md-regular",
  md: "xl-regular",
  lg: "2xl-regular",
} as const;

export function RegionFlag({
  flagCode,
  size = "md",
  shape = "rounded",
  className,
}: RegionFlagProps) {
  const isLocal = flagCode === "local";

  // "square" shows the real, rectangular country flag at its natural aspect —
  // no container, no crop. Local renders the glyph in a matching 4:3 frame.
  if (shape === "square") {
    if (isLocal) {
      return (
        <div
          className={cn(
            "flex aspect-[4/3] items-center justify-center rounded-[3px] bg-grayA-3 text-gray-11",
            rectHeight[size],
            className,
          )}
        >
          <Laptop2 iconSize={localIconSize[size]} />
        </div>
      );
    }
    return (
      <img
        src={`/images/flags/${flagCode}.svg`}
        alt={flagCode}
        className={cn("w-auto", rectHeight[size], className)}
      />
    );
  }

  // circle / rounded: contents fill a fixed square box and are clipped to shape.
  return (
    <div
      className={cn(
        "bg-grayA-3 flex items-center justify-center overflow-hidden text-gray-11",
        sizeClass[size],
        shapeClass[shape],
        shape === "rounded" && "border border-grayA-3",
        className,
      )}
    >
      {isLocal ? (
        <Laptop2 iconSize={localIconSize[size]} />
      ) : (
        <img
          src={`/images/flags/${flagCode}.svg`}
          alt={flagCode}
          className="size-full object-cover"
        />
      )}
    </div>
  );
}
