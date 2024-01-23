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
      <dl className="grid grid-cols-1 gap-10 sm:grid-cols-2 lg:auto-cols-fr lg:grid-flow-col lg:grid-cols-none">
        {children}
      </dl>
    </FadeInStagger>
  );
}

export function StatListItem({
  label,
  value,
}: {
  label?: string;
  value?: string;
}) {
  return (
    <Border as={FadeIn} position="left" className="flex flex-col-reverse pl-8">
      <dt className="mt-2 text-base text-white">{label}</dt>
      <dd className="text-3xl font-semibold font-display text-white sm:text-4xl">{value}</dd>
    </Border>
  );
}
