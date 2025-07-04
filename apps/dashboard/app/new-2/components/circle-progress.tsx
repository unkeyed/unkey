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
  const isComplete = value >= total;

  if (isComplete) {
    return (
      <div className={cn("inline-flex items-center justify-center", className)}>
        <svg
          width={size}
          height={size}
          viewBox="0 0 18 18"
          className="text-success-9"
        >
          <g fill="currentColor" strokeLinecap="butt" strokeLinejoin="miter">
            <path
              d="M9 1.5a7.5 7.5 0 1 0 0 15 7.5 7.5 0 1 0 0-15z"
              fill="none"
              stroke="currentColor"
              strokeLinecap="square"
              strokeMiterlimit="10"
              strokeWidth={strokeWidth}
            />
            <path
              d="M5.25 9.75l2.25 2.25 5.25-6"
              fill="none"
              stroke="currentColor"
              strokeLinecap="square"
              strokeMiterlimit="10"
              strokeWidth={strokeWidth}
            />
          </g>
        </svg>
      </div>
    );
  }

  const radius = (size - strokeWidth) / 2;
  const circumference = radius * 2 * Math.PI;
  const progress = Math.min((value / total) * 100, 100);
  const strokeDasharray = circumference;
  const strokeDashoffset = circumference - (progress / 100) * circumference;

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
            transition: "stroke-dashoffset 0.3s ease-in-out",
          }}
        />
      </svg>
    </div>
  );
};
