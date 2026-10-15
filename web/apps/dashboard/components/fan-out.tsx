import { cn } from "cn";

type FanOutProps = {
  targets?: number;
  className?: string;
};

export function FanOut({ targets = 1, className }: FanOutProps) {
  return (
    <svg
      viewBox="0 0 100 100"
      preserveAspectRatio="none"
      className={cn("pointer-events-none", className)}
      aria-hidden="true"
    >
      {Array.from({ length: targets }, (_, i) => ((i + 0.5) / targets) * 100).map((x) => (
        <path
          key={x}
          d={`M50,0 V50 H${x} V100`}
          fill="none"
          strokeWidth={1}
          strokeDasharray="3 3"
          vectorEffect="non-scaling-stroke"
          className="animate-dash-flow stroke-gray-7 motion-reduce:animate-none"
        />
      ))}
    </svg>
  );
}
