"use client";
import { CloudUp, Harddrive, HeartPulse, Location2, Nodes2 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";

type IconBoxProps = {
  children?: React.ReactNode;
  large?: boolean;
  className?: string;
};

const IconBox = ({ children, large, className }: IconBoxProps) => (
  <div
    className={cn(
      "shrink-0 flex items-center justify-center rounded-[10px] bg-transparent ring-1 ring-grayA-4 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.12),0_0_0_0.75px_rgba(0,0,0,0.08)]",
      large ? "size-16" : "size-9",
      className,
    )}
  >
    {children}
  </div>
);

const items: { icon: React.ReactNode; large?: boolean; opacity: string }[] = [
  { icon: null, opacity: "opacity-60" },
  { icon: <Harddrive className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <Location2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <CloudUp className="size-9" iconSize="md-thin" />, large: true, opacity: "opacity-90" },
  { icon: <HeartPulse className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <Nodes2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: null, opacity: "opacity-60" },
];

export const Onboarding = () => (
  <div className="flex flex-col items-center justify-center h-screen">
    <div
      className="p-2"
      style={{
        maskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
        WebkitMaskImage:
          "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
      }}
    >
      <div className="flex gap-6 items-center justify-center text-gray-12">
        {items.map((item, i) => (
          <IconBox key={i} large={item.large} className={item.opacity}>
            {item.icon}
          </IconBox>
        ))}
      </div>
    </div>

    <div className="mb-5" />
    {/* Title-desc */}
    <div className="flex flex-col items-center justify-center gap-2" />
    <div className="mb-6" />
    {/* actions */}
    <div className="mb-7" />
    {/* sub-actions */}
  </div>
);
