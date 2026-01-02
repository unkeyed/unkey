// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { cn } from "../lib/utils";

type SettingCardProps = {
  title: string | React.ReactNode;
  description: string | React.ReactNode;
  children?: React.ReactNode;
  className?: string;
  border?: "top" | "bottom" | "both" | "none" | "default";
  contentWidth?: string;
};

function SettingCard({
  title,
  description,
  children,
  className,
  border = "default",
  contentWidth = "w-[420px]",
}: SettingCardProps) {
  const borderRadiusClass = {
    "rounded-t-xl": border === "top",
    "rounded-b-xl": border === "bottom",
    "rounded-xl": border === "both",
    "": border === "none" || border === "default",
  };

  const borderClass = {
    "border border-grayA-4": border !== "none",
    "border-t-0": border === "bottom",
    "border-b-0": border === "top",
  };

  return (
    <div
      className={cn(
        "px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row ",
        borderRadiusClass,
        borderClass,
        className,
      )}
    >
      <div className="flex flex-col gap-1 text-sm w-fit">
        <div className="font-medium text-accent-12 leading-5 tracking-normal">{title}</div>
        <div className="font-normal text-accent-11 text-[13px] leading-5 tracking-normal">
          {description}
        </div>
      </div>
      <div className={cn("flex w-full", contentWidth)}>{children}</div>
    </div>
  );
}

SettingCard.displayName = "SettingCard";

export { SettingCard };
