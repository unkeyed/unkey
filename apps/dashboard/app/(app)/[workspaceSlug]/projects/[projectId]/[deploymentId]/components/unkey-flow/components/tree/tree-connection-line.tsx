import { useRef, useState, useMemo, useEffect } from "react";
import type { Point } from "../../types";

const STRAIGHT_LINE_THRESHOLD = 20;
const MIN_VERTICAL_FOR_CURVE = 100;
const CORNER_RADIUS = 32;
const MIN_CORNER_SPACING = 64;
const STROKE_WIDTH_BASE = 2.5;
const STROKE_WIDTH_ANIMATED = 3;
const LIGHT_BAND_SIZE = 40;
const ANIMATION_VELOCITY = 100;
const ANIMATION_OPACITY = 0.94;

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
    if (Math.abs(dx) < STRAIGHT_LINE_THRESHOLD || dy < MIN_VERTICAL_FOR_CURVE) {
      return `M ${from.x} ${from.y} L ${to.x} ${to.y}`;
    }

    // Curved path with rounded corners
    const midY = from.y + dy * 0.5;
    const y1 = Math.min(
      midY - CORNER_RADIUS,
      from.y + dy - MIN_CORNER_SPACING - CORNER_RADIUS
    );
    const direction = dx > 0 ? 1 : -1;
    const corner1Y = y1 + CORNER_RADIUS;
    const corner1X = from.x + CORNER_RADIUS * direction;
    const corner2X = to.x - CORNER_RADIUS * direction;
    const corner2Y = to.y - CORNER_RADIUS;

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

  const gapSize = pathLength - LIGHT_BAND_SIZE;
  const dashArray = `${LIGHT_BAND_SIZE} ${gapSize}`;
  const duration = pathLength / ANIMATION_VELOCITY;

  return (
    <>
      <path
        ref={pathRef}
        d={pathD}
        className="stroke-gray-3"
        strokeWidth={STROKE_WIDTH_BASE}
        fill="none"
        strokeLinecap="round"
      />
      <path
        d={pathD}
        className="stroke-grayA-12"
        strokeWidth={STROKE_WIDTH_ANIMATED}
        fill="none"
        strokeLinecap="round"
        strokeDasharray={dashArray}
        strokeDashoffset={pathLength}
        style={{ opacity: ANIMATION_OPACITY }}
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
