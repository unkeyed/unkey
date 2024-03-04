import { Border } from "@/components/border";
import { FadeIn, FadeInStagger } from "@/components/fade-in";
import { NumberTicker } from "@/components/number-ticker";
import { cn } from "@/lib/utils";
import React from "react";

export function StatList({
  children,
  ...props
}: {
  children: React.ReactNode;
  props?: any;
}) {
  return (
    <dl className="grid grid-cols-2 md:grid-flow-col md:grid-cols-none items-center">{children}</dl>
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
      position="left"
      className={cn(
        "flex-col-reverse pl-8 border-white/[.15] border-l max-w-[200px] md:mb-0",
        className,
      )}
    >
      <div>
        <dd className="font-semibold font-display stats-number-gradient text-4xl">
          <NumberTicker value={value} />
        </dd>
        <dt className="mt-2 text-white/50">{label}</dt>
      </div>
    </Border>
  );
}
