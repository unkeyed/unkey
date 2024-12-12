import * as React from "react";

interface MetricProps {
  label: string;
  value: string | React.ReactNode;
}

const Metric = React.forwardRef<HTMLDivElement, MetricProps>(({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-center px-4 py-2 border-border">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
    </div>
  );
});

export { Metric };
