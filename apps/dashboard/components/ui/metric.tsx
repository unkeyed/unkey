import { cn } from "@/lib/utils";
import type * as React from "react";

interface MetricProps {
  label: string;
  value: string | React.ReactNode;
  className?: string;
}

const Metric: React.FC<MetricProps> = ({ label, value, className }) => {
  return (
    <div
      className={cn(
        "flex flex-col items-start justify-center px-4 py-2 border-border gap-1",
        className,
      )}
    >
      <p className="text-sm text-accent-11">{label}</p>
      <div className="text-lg font-medium leading-none tracking-tight text-accent-12">{value}</div>
    </div>
  );
};

export { Metric };
