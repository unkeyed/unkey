import { useRef, useState, useMemo, useEffect } from "react";
import type { Point } from "../../types";

/**
 * Animated connection line between two points in vertical tree.
 * Draws curved path for horizontal separation, straight line otherwise.
 * Includes animated light band traveling along the path.
 * @throws Error if path ref is null or path length is zero
 */
type TreeConnectionLineProps = {
  from: Point;
  to: Point;
};

export function TreeConnectionLine({ from, to }: TreeConnectionLineProps) {
  const pathRef = useRef<SVGPathElement>(null);
  const [pathLength, setPathLength] = useState(200);

  const pathD = useMemo(() => {
    const dx = to.x - from.x;
    const dy = to.y - from.y;

    // Straight line for short distances or minimal horizontal offset
    if (Math.abs(dx) < 20 || dy < 100) {
      return `M ${from.x} ${from.y} L ${to.x} ${to.y}`;
    }

    // Curved path with rounded corners
    const radius = 32;
    const midY = from.y + dy * 0.5;
    const y1 = Math.min(midY - radius, from.y + dy - 64 - radius);
    const direction = dx > 0 ? 1 : -1;

    const corner1Y = y1 + radius;
    const corner1X = from.x + radius * direction;
    const corner2X = to.x - radius * direction;
    const corner2Y = to.y - radius;

    return `M ${from.x} ${from.y} L ${from.x} ${y1} Q ${from.x} ${corner1Y} ${corner1X} ${corner1Y} L ${corner2X} ${corner1Y} Q ${to.x} ${corner1Y} ${to.x} ${corner2Y} L ${to.x} ${to.y}`;
  }, [from, to]);

  useEffect(() => {
    if (!pathRef.current) {
      throw new Error("Failed to get SVG path ref for connection line");
    }
    const length = pathRef.current.getTotalLength();
    if (length === 0) {
      throw new Error(
        `Connection line path has zero length for ${from.x},${from.y} -> ${to.x},${to.y}`
      );
    }
    setPathLength(length);
  }, [pathD, from, to]);

  const lightBandSize = 40;
  const velocity = 100;
  const gapSize = pathLength - lightBandSize;
  const dashArray = `${lightBandSize} ${gapSize}`;
  const duration = pathLength / velocity;

  return (
    <>
      <path
        ref={pathRef}
        d={pathD}
        className="stroke-gray-3"
        strokeWidth="2.5"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d={pathD}
        className="stroke-grayA-12"
        strokeWidth="3"
        fill="none"
        strokeLinecap="round"
        strokeDasharray={dashArray}
        strokeDashoffset={pathLength}
        style={{ opacity: 0.94 }}
      >
        <animate
          attributeName="stroke-dashoffset"
          from={pathLength}
          to={0}
          dur={`${duration}s`}
          repeatCount="indefinite"
        />
      </path>
    </>
  );
}
