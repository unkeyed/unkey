import { Border } from "@/components/border";
import { FadeIn, FadeInStagger } from "@/components/fade-in";
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
    <FadeInStagger {...props}>
      <dl className="grid grid-cols-2 md:grid-flow-col md:grid-cols-none items-center">
        {children}
      </dl>
    </FadeInStagger>
  );
}

type ParsedData = {
  value: number;
  unit?: string;
};

function parseData(input: string): ParsedData {
  const match = input.match(/^(\d+(?:\.\d+)?)([kmb]?)$/i);
  if (match) {
    return { value: parseFloat(match[1]), unit: match[2] || undefined };
  }
  throw new Error("Invalid input format");
}

export function StatListItem({
  label,
  value,
  className,
}: {
  label: string;
  value: string;
  className?: string;
}) {
  const data = parseData(value);
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
          {data.value} {data.unit}
        </dd>
        <dt className="mt-2 text-white/50">{label}</dt>
      </div>
    </Border>
  );
}
