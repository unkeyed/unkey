// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import type { SVGProps } from "react";
import { AnimatedLoadingSpinner } from "./animated-loading-spinner";
import { cn } from "../lib/utils";

interface LoadingProps extends SVGProps<SVGSVGElement> {
  /** Animation duration, e.g. "124ms" per segment  for spinner type or "0.75s" for dots type */
  duration?: string;
  size?: number;
  type?: "spinner" | "dots";
}

function Loading({ size, duration, className, type = "spinner", ...props }: LoadingProps) {
  if (type === "dots") {
    return <DotsLoading size={size ?? 24} className={className} duration={duration} {...props} />;
  }

  return (
    <AnimatedLoadingSpinner
      size={size ?? 18}
      segmentTimeInMS={duration ? convertDurationToMS(duration) : undefined}
      className={cn("fill-current", className)}
      {...props}
    />
  );
}

type DotsLoadingProps = SVGProps<SVGSVGElement> & {
  size?: number;
  duration?: string;
};

const DotsLoading = ({ size, className, duration = "0.75s", ...props }: DotsLoadingProps) => {
  return (
    <svg
      {...props}
      className={cn("fill-current", className)}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="4" cy="12" r="3">
        <animate id="a" begin="0;b.end-0.25s" attributeName="r" dur={duration} values="3;.2;3" />
      </circle>
      <circle cx="12" cy="12" r="3">
        <animate begin="a.end-0.6s" attributeName="r" dur={duration} values="3;.2;3" />
      </circle>
      <circle cx="20" cy="12" r="3">
        <animate id="b" begin="a.end-0.45s" attributeName="r" dur={duration} values="3;.2;3" />
      </circle>
    </svg>
  );
};

const convertDurationToMS = (duration: string): number => {
  const lowerDuration = duration.toLowerCase().trim();

  if (lowerDuration.endsWith("ms")) {
    const msValue = Number.parseFloat(lowerDuration.slice(0, -2));
    if (Number.isNaN(msValue)) {
      throw new Error(`Invalid milliseconds format: "${duration}"`);
    }
    return msValue;
  }

  if (lowerDuration.endsWith("s")) {
    const secondsValue = Number.parseFloat(lowerDuration.slice(0, -1));
    if (Number.isNaN(secondsValue)) {
      throw new Error(`Invalid seconds format: "${duration}"`);
    }
    return secondsValue * 1000;
  }

  throw new Error(`Invalid duration format: "${duration}". Expected 'ms' or 's' suffix.`);
};

Loading.displayName = "Loading";

export { Loading };
