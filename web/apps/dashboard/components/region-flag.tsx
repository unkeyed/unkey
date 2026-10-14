import { type RegionInfo, regionInfo } from "@/lib/regions";
import BoringAvatar from "boring-avatars";
import { cn } from "cn";

type Shape = "circle" | "rounded" | "rect";
type Size = "xs" | "sm" | "md" | "lg";

const SHAPES: Record<
  Shape,
  { frame: string; art: string; container: Record<Size, string>; glyph: Record<Size, string> }
> = {
  circle: {
    frame: "rounded-full bg-grayA-3",
    art: "/images/flags",
    container: { xs: "size-4", sm: "size-[22px]", md: "size-9", lg: "size-12" },
    glyph: { xs: "size-4", sm: "size-4", md: "size-4", lg: "size-[22px]" },
  },
  rounded: {
    frame: "rounded-xl border bg-grayA-3",
    art: "/images/flags",
    container: { xs: "size-4", sm: "size-[22px]", md: "size-9", lg: "size-12" },
    glyph: { xs: "size-4", sm: "size-4", md: "size-4", lg: "size-[22px]" },
  },
  rect: {
    frame: "overflow-hidden bg-grayA-3 ring-1 ring-grayA-6",
    art: "/images/flags/rect",
    container: { xs: "h-3 w-4", sm: "h-3.5 w-5", md: "h-5 w-7", lg: "h-7 w-10" },
    glyph: { xs: "size-full", sm: "size-full", md: "size-full", lg: "size-full" },
  },
};

type RegionFlagProps = {
  region: string;
  size?: Size;
  shape?: Shape;
  className?: string;
};

export function RegionFlag({ region, size = "md", shape = "rounded", className }: RegionFlagProps) {
  const { frame, container } = SHAPES[shape];
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center",
        frame,
        container[size],
        className,
      )}
    >
      <FlagArt flag={regionInfo(region).flag} shape={shape} size={size} />
    </span>
  );
}

function FlagArt({ flag, shape, size }: { flag: RegionInfo["flag"]; shape: Shape; size: Size }) {
  if (flag === null || (flag === "local" && shape === "rect")) {
    return null;
  }
  if (flag === "local") {
    return <BoringAvatar size="100%" variant="marble" />;
  }
  const { art, glyph } = SHAPES[shape];
  return <img src={`${art}/${flag}.svg`} alt={flag} className={glyph[size]} />;
}
