import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

type GlowIconProps = {
  icon: ReactNode;
  variant?: "feature" | "error";
  glow?: boolean;
  transition?: boolean;
  className?: string;
};

export function GlowIcon({
  icon,
  variant = "feature",
  glow = true,
  transition = false,
  className,
}: GlowIconProps) {
  const glowColor =
    variant === "error"
      ? "bg-linear-to-l from-error-7 to-error-8"
      : "bg-linear-to-l from-feature-8 to-info-9";

  const glowVisible = transition
    ? glow
      ? "animate-pulse opacity-20"
      : "opacity-0 transition-opacity duration-300"
    : glow
      ? "animate-pulse opacity-20"
      : "hidden";

  const iconBg =
    variant === "error"
      ? "bg-errorA-3 dark:text-error-11 text-error-11"
      : glow
        ? "dark:bg-white dark:text-black bg-black text-white shadow-md shadow-black/40"
        : "";

  return (
    <div className={cn("relative", className)}>
      <div
        className={cn("absolute inset-[-4px] rounded-[10px] blur-[14px]", glowColor, glowVisible)}
      />
      <div
        className={cn(
          "relative w-full h-full rounded-[10px] flex items-center justify-center shrink-0",
          iconBg,
        )}
      >
        {icon}
      </div>
    </div>
  );
}
