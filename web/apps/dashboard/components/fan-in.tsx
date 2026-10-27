import { cn } from "cn";

type FanInProps = {
  className?: string;
};

const W = 480;
const H = 120;
const MEET = H - 34;
const RADII = [44, 100, 160, 226];

export function FanIn({ className }: FanInProps) {
  const cx = W / 2;
  return (
    <svg
      viewBox={`0 0 ${W} ${H}`}
      preserveAspectRatio="xMidYMax meet"
      className={cn("pointer-events-none fill-none stroke-gray-7", className)}
      aria-hidden="true"
    >
      <g className="[mask-image:linear-gradient(to_bottom,transparent,#000_60%)]">
        {RADII.map((rx) => (
          <path
            key={rx}
            d={`M${cx - rx},0 A${rx},${MEET} 0 0 0 ${cx},${MEET} A${rx},${MEET} 0 0 0 ${cx + rx},0`}
            strokeWidth={1}
            vectorEffect="non-scaling-stroke"
          />
        ))}
      </g>
      <path
        d={`M${cx},${MEET} V${H}`}
        strokeWidth={1}
        strokeDasharray="3 3"
        vectorEffect="non-scaling-stroke"
        className="animate-dash-flow motion-reduce:animate-none"
      />
    </svg>
  );
}
