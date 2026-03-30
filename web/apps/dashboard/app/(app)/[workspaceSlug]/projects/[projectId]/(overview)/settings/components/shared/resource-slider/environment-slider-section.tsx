import type { ReactNode } from "react";

type EnvironmentSliderSectionProps = {
  label: string;
  children: ReactNode;
};

export const EnvironmentSliderSection = ({ label, children }: EnvironmentSliderSectionProps) => (
  <div className="flex flex-col mb-4">
    <span className="text-gray-11 text-[13px] mb-1">{label}</span>
    {children}
  </div>
);
