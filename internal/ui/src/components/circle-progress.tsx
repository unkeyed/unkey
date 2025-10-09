// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { type VariantProps, cva } from "class-variance-authority";
import { forwardRef } from "react";
import { cn } from "../lib/utils";
import { type iconSize as IconSize, sizeMap } from "@unkey/icons";

const circleProgressVariants = cva("inline-flex items-center justify-center", {
  variants: {
    variant: {
      primary:
        "[&_.progress-circle]:text-grayA-9 [&_.complete-circle]:text-successA-9 [&_.complete-check]:text-successA-9",
      secondary:
        "[&_.progress-circle]:text-grayA-7 [&_.complete-circle]:text-grayA-9 [&_.complete-check]:text-grayA-9",
      success:
        "[&_.progress-circle]:text-successA-9 [&_.complete-circle]:text-successA-9 [&_.complete-check]:text-successA-9",
      warning:
        "[&_.progress-circle]:text-warningA-9 [&_.complete-circle]:text-warningA-9 [&_.complete-check]:text-warningA-9",
      error:
        "[&_.progress-circle]:text-errorA-9 [&_.complete-circle]:text-errorA-9 [&_.complete-check]:text-errorA-9",
    },
  },
  defaultVariants: {
    variant: "primary",
  },
});

type CircleProgressProps = React.HTMLAttributes<HTMLDivElement> & {
  /** Current progress value (e.g., 3 completed items out of 5 total) */
  value: number;
  /** Total/maximum value that represents 100% completion */
  total: number;
  /**
   * Size using the standard icon sizing system (size-weight format)
   * @default "md-regular"
   * @example "sm-thin", "lg-bold", "xl-medium"
   */
  iconSize?: IconSize;
} & VariantProps<typeof circleProgressVariants>;

/**
 * CircleProgress - A circular progress indicator with completion checkmark
 *
 * Shows progress as a circular arc that fills clockwise. When value >= total,
 * displays a checkmark instead of the progress arc with a smooth transition.
 * Uses the same sizing system as icons for consistency.
 *
 * @example
 * ```tsx
 * // Basic usage - shows 3/5 progress
 * <CircleProgress value={3} total={5} />
 *
 * // With custom styling and size matching icon system
 * <CircleProgress
 *   value={validFields}
 *   total={requiredFields}
 *   variant="success"
 *   iconSize="lg-medium"
 * />
 *
 * // Small thin progress indicator
 * <CircleProgress
 *   value={progress}
 *   total={100}
 *   iconSize="sm-thin"
 * />
 * ```
 */
export const CircleProgress = forwardRef<HTMLDivElement, CircleProgressProps>(
  (
    { value, total, iconSize = "md-regular", className, variant, ...props },
    ref
  ) => {
    // Early error validation
    if (total <= 0) {
      throw new Error("CircleProgress: total must be greater than 0");
    }
    if (value < 0) {
      throw new Error("CircleProgress: value cannot be negative");
    }
    if (!Number.isFinite(value) || !Number.isFinite(total)) {
      throw new Error("CircleProgress: value and total must be finite numbers");
    }

    const { iconSize: actualSize, strokeWidth } = sizeMap[iconSize];

    const radius = (actualSize - strokeWidth) / 2;
    const circumference = radius * 2 * Math.PI;
    const progress = Math.min((value / total) * 100, 100);
    const strokeDasharray = circumference;
    const strokeDashoffset = circumference - (progress / 100) * circumference;
    const isComplete = value >= total;

    return (
      <div
        ref={ref}
        className={cn(circleProgressVariants({ variant }), className)}
        {...props}
      >
        <svg
          width={actualSize}
          height={actualSize}
          className="transform -rotate-90"
        >
          {/* Background circle */}
          <circle
            cx={actualSize / 2}
            cy={actualSize / 2}
            r={radius}
            className="text-grayA-6"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            fill="none"
          />
          {/* Progress circle */}
          <circle
            cx={actualSize / 2}
            cy={actualSize / 2}
            r={radius}
            className="progress-circle"
            strokeWidth={strokeWidth}
            stroke="currentColor"
            fill="none"
            strokeDasharray={strokeDasharray}
            strokeDashoffset={strokeDashoffset}
            strokeLinecap="round"
            style={{
              transition:
                "stroke-dashoffset 0.3s ease-in-out, opacity 0.2s ease-in-out",
              opacity: isComplete ? 0 : 1,
            }}
          />
          {/* Checkmark group */}
          <g
            style={{
              transition: "opacity 0.2s ease-in-out",
              opacity: isComplete ? 1 : 0,
            }}
          >
            {/* Circle background for checkmark */}
            <circle
              cx={actualSize / 2}
              cy={actualSize / 2}
              r={radius}
              fill="none"
              stroke="currentColor"
              strokeWidth={strokeWidth}
              strokeLinecap="square"
              className="complete-circle"
            />
            {/* Checkmark path */}
            <path
              d={`M${actualSize * 0.292} ${actualSize * 0.542} l${
                actualSize * 0.125
              } ${actualSize * 0.125} l${actualSize * 0.292} -${
                actualSize * 0.333
              }`}
              fill="none"
              stroke="currentColor"
              strokeWidth={strokeWidth}
              strokeLinecap="square"
              transform={`rotate(90 ${actualSize / 2} ${actualSize / 2})`}
              className="complete-check"
            />
          </g>
        </svg>
      </div>
    );
  }
);

CircleProgress.displayName = "CircleProgress";

export { circleProgressVariants };
export type { CircleProgressProps };
