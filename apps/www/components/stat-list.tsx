import { Border } from "@/components/border";
import { FadeIn } from "@/components/fade-in";
import { NumberTicker } from "@/components/number-ticker";
import { cn } from "@/lib/utils";
import type React from "react";

export function StatList({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <dl className="grid items-center grid-cols-2 gap-8 md:grid-flow-col md:grid-cols-none">
      {children}
    </dl>
  );
}

export function StatListItem({
  label,
  value,
  className,
}: {
  label: string;
  value: number;
  className?: string;
}) {
  return (
    <Border
      as={FadeIn}
      className={cn(
        "flex-col-reverse pl-6 sm:pl-8 border-white/[.15] border-l max-w-[200px] md:mb-0",
        className,
      )}
    >
      <dd className="text-4xl font-semibold font-display stats-number-gradient w-max">
        <NumberTicker value={value} />
      </dd>
      <dt className="mt-2 text-base font-light text-white/50">{label}</dt>
    </Border>
  );
}
