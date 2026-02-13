import type { FlagCode } from "@/lib/trpc/routers/deploy/network/utils";
import { cn } from "@/lib/utils";

type RegionFlagSize = "xs" | "sm" | "md" | "lg";
type RegionFlagShape = "rounded-sm" | "circle";

type RegionFlagProps = {
  flagCode: FlagCode;
  size?: RegionFlagSize;
  shape?: RegionFlagShape;
  className?: string;
};

const sizeConfig = {
  xs: {
    container: "size-4",
    flag: "size-4",
    padding: "",
  },
  sm: {
    container: "size-[22px]",
    flag: "size-4",
    padding: "p-[3px]",
  },
  md: {
    container: "size-9",
    flag: "size-4",
    padding: "",
  },
  lg: {
    container: "size-12",
    flag: "size-[22px]",
    padding: "",
  },
};

const shapeClass = {
  rounded: "rounded-[10px]",
  circle: "rounded-full",
};

export function RegionFlag({
  flagCode,
  size = "md",
  shape = "rounded-sm",
  className,
}: RegionFlagProps) {
  const config = sizeConfig[size];
  const hasExplicitPadding = config.padding !== "";

  return (
    <div
      className={cn(
        "bg-grayA-3 flex items-center justify-center",
        config.container,
        shapeClass[shape],
        shape === "rounded-sm" && "border border-grayA-3",
        hasExplicitPadding && config.padding,
        !hasExplicitPadding && "p-0",
        className,
      )}
    >
      <img src={`/images/flags/${flagCode}.svg`} alt={flagCode} className={config.flag} />
    </div>
  );
}
