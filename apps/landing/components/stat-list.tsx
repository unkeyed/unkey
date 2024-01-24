import { Border } from "@/components/border";
import { FadeIn, FadeInStagger } from "@/components/fade-in";
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
      <dl className="grid grid-cols-1 gap-10 grid-cols-2 lg:auto-cols-fr lg:grid-flow-col lg:grid-cols-none">
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
  } else {
    throw new Error("Invalid input format");
  }
}

export function StatListItem({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  console.log({ value });
  const data = parseData(value);
  return (
    <Border as={FadeIn} position="left" className="flex flex-col-reverse pl-8">
      <div>
        <dd className="font-semibold font-display stats-number-gradient text-4xl">
          {data.value} {data.unit}
        </dd>
        <dt className="mt-2 text-white/50">{label}</dt>
      </div>
    </Border>
  );
}
