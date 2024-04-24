import type React from "react";

import { cn } from "@/lib/utils";
type SectionHeaderProps = {
  tag: string;
  title: string;
  subtitle: string;
  actions?: React.ReactNode[];
  align?: "left" | "center";
};
export const SectionHeader: React.FC<SectionHeaderProps> = ({
  tag,
  title,
  subtitle,
  actions,
  align,
}): JSX.Element => {
  return (
    <div
      className={cn("flex flex-col gap-8  max-w-3xl mx-auto", {
        "items-center text-center": !align || align === "center",
        "items-start text-left": align === "left",
      })}
    >
      <span className="font-mono text-sm text-white/50">{tag}</span>
      <h2 className="bg-gradient-to-r text-transparent bg-clip-text from-white via-white via-40% to-[#4C4C4C] text-6xl font-medium">
        {title}
      </h2>

      <p className="text-sm text-white/80">{subtitle}</p>

      <div className="flex items-center gap-4">{actions}</div>
    </div>
  );
};
