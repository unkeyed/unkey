// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React, { useId } from "react";
import type { SVGProps } from "react";

interface LoadingProps extends SVGProps<SVGSVGElement> {
  /** Animation duration, e.g. "0.75s" */
  dur?: string;
}

export function Loading({
  width = 24,
  height = 24,
  dur = "0.75s",
}: LoadingProps): JSX.Element {
  const id = useId();

  return (
    <svg
      className="fill-current"
      width={width}
      height={height}
      viewBox="0 0 24 24"
      aria-label="Loading"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="4" cy="12" r="3">
        <animate
          id={`${id}-a`}
          begin={`0;${id}-b.end-0.25s`}
          attributeName="r"
          dur={dur}
          values="3;.2;3"
        />
      </circle>
      <circle cx="12" cy="12" r="3">
        <animate
          begin={`${id}-a.end-0.6s`}
          attributeName="r"
          dur={dur}
          values="3;.2;3"
        />
      </circle>
      <circle cx="20" cy="12" r="3">
        <animate
          id={`${id}-b`}
          begin={`${id}-a.end-0.45s`}
          attributeName="r"
          dur={dur}
          values="3;.2;3"
        />
      </circle>
    </svg>
  );
}
