import { cn } from "@unkey/ui/src/lib/utils";

type CircleProgressProps = {
  value: number;
  total: number;
  size?: number;
  strokeWidth?: number;
  className?: string;
};

export const CircleProgress = ({
  value,
  total,
  size = 20,
  strokeWidth = 2,
  className = "",
}: CircleProgressProps) => {
  const radius = (size - strokeWidth) / 2;
  const circumference = radius * 2 * Math.PI;
  const progress = Math.min((value / total) * 100, 100);
  const strokeDasharray = circumference;
  const strokeDashoffset = circumference - (progress / 100) * circumference;
  const isComplete = value >= total;

  return (
    <div className={cn("inline-flex items-center justify-center", className)}>
      <svg width={size} height={size} className="transform -rotate-90">
        {/* Background circle */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          className="text-gray-6"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          fill="none"
        />
        {/* Progress circle */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          className="text-success-9"
          strokeWidth={strokeWidth}
          stroke="currentColor"
          fill="none"
          strokeDasharray={strokeDasharray}
          strokeDashoffset={strokeDashoffset}
          strokeLinecap="round"
          style={{
            transition: "stroke-dashoffset 0.3s ease-in-out, opacity 0.2s ease-in-out",
            opacity: isComplete ? 0 : 1,
          }}
        />
        {/* Checkmark group */}
        <g
          className="text-success-9"
          style={{
            transition: "opacity 0.2s ease-in-out",
            opacity: isComplete ? 1 : 0,
          }}
        >
          {/* Circle background for checkmark */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            strokeLinecap="square"
          />
          {/* Checkmark path */}
          <path
            d={`M${size * 0.292} ${size * 0.542} l${size * 0.125} ${
              size * 0.125
            } l${size * 0.292} -${size * 0.333}`}
            fill="none"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            strokeLinecap="square"
            transform={`rotate(90 ${size / 2} ${size / 2})`}
          />
        </g>
      </svg>
    </div>
  );
};
