import {
  Code,
  CloudUp,
  Cube,
  Earth,
  Github,
  Harddrive,
  HeartPulse,
  Location2,
  Nodes2,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";

// Classic "old style" artwork: a flanked row of bordered icon boxes with the cloud as the focal
// point. Restored from the pre-halftone onboarding header so it can be chosen via ArtStyleSwitcher.
type IconBoxProps = {
  children?: ReactNode;
  large?: boolean;
  className?: string;
};

const IconBox = ({ children, large, className }: IconBoxProps) => (
  <div
    className={cn(
      "shrink-0 flex items-center justify-center rounded-[10px] bg-transparent ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none",
      large ? "size-16" : "size-9",
      className,
    )}
  >
    {children}
  </div>
);

const iconItems: { icon: ReactNode; large?: boolean; opacity: string }[] = [
  { icon: null, opacity: "opacity-60" },
  { icon: <Harddrive className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <Location2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <CloudUp className="size-9" iconSize="md-thin" />, large: true, opacity: "opacity-90" },
  { icon: <HeartPulse className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <Nodes2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: null, opacity: "opacity-60" },
];

// Projects empty state: the project box (center, focal) with repo (GitHub) and build (code) as its
// immediate flanks, then regions (globe) and liveness (pulse) on the outer edges. Boxes dim toward
// the edges and the row fades out at both ends so the box stays the focus.
const projectFlank: { icon: ReactNode; large?: boolean; opacity: string }[] = [
  { icon: <Earth className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-50" },
  { icon: <Github className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <Cube className="size-9" iconSize="md-thin" />, large: true, opacity: "opacity-90" },
  { icon: <Code className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <HeartPulse className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-50" },
];

export const ClassicProjectIcon = ({ className }: { className?: string }) => (
  <div
    className={cn("p-2", className)}
    style={{
      maskImage: "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
      WebkitMaskImage: "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
    }}
  >
    <div className="flex gap-6 items-center justify-center text-gray-12">
      {projectFlank.map((item, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: static row, index is stable
        <IconBox key={i} large={item.large} className={item.opacity}>
          {item.icon}
        </IconBox>
      ))}
    </div>
  </div>
);

export const ClassicIconRow = ({ className }: { className?: string }) => (
  <div
    className={cn("p-2", className)}
    style={{
      maskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
      WebkitMaskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
    }}
  >
    <div className="flex gap-6 items-center justify-center text-gray-12">
      {iconItems.map((item, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: static row, index is stable
        <IconBox key={i} large={item.large} className={item.opacity}>
          {item.icon}
        </IconBox>
      ))}
    </div>
  </div>
);
