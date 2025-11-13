import { useRef, useState, useMemo, useEffect } from "react";
import type { Point } from "../../types";
import { renderPath, move, line, curve } from "./tree-path-command";

const PATH_CONFIG = {
  straightLineThreshold: 20,
  minVerticalForCurve: 100,
  cornerRadius: 32,
  minCornerSpacing: 64,
} as const;

const ANIMATION_CONFIG = {
  strokeWidthBase: 2.5,
  strokeWidthAnimated: 3,
  lightBandSize: 40,
  defaultVelocity: 100,
  opacity: 0.94,
} as const;

type TreeConnectionLineProps = {
  from: Point;
  to: Point;
  horizontal?: boolean;
  animationVelocity?: number;
};

/**
 * Animated connection line between two points in a tree layout.
 * Draws curved paths for visual clarity, straight lines when points are close.
 * Includes animated light band traveling along the path.
 */
export function TreeConnectionLine({
  from,
  to,
  horizontal = false,
  animationVelocity = ANIMATION_CONFIG.defaultVelocity,
}: TreeConnectionLineProps) {
  const pathRef = useRef<SVGPathElement>(null);
  const [pathLength, setPathLength] = useState(200);

  const pathD = useMemo(() => {
    if (horizontal) {
      const dy = Math.abs(to.y - from.y);
      return dy < PATH_CONFIG.cornerRadius * 2
        ? straightLine(from, to)
        : steppedHorizontal(from, to);
    }

    // Vertical layout with curves
    const dx = Math.abs(to.x - from.x);
    const dy = to.y - from.y;
    const needsStraightLine =
      dx < PATH_CONFIG.straightLineThreshold ||
      dy < PATH_CONFIG.minVerticalForCurve;

    return needsStraightLine ? straightLine(from, to) : roundedZShape(from, to);
  }, [from, to, horizontal]);

  useEffect(() => {
    const path = pathRef.current;
    if (!path) {
      console.error("SVG path ref is null");
      return;
    }

    const length = path.getTotalLength();
    if (length === 0) {
      console.error(`Zero-length path: ${from.x},${from.y} -> ${to.x},${to.y}`);
      return;
    }

    setPathLength(length);
  }, [pathD, from, to]);

  const gapSize = pathLength - ANIMATION_CONFIG.lightBandSize;
  const dashArray = `${ANIMATION_CONFIG.lightBandSize} ${gapSize}`;
  const duration = pathLength / animationVelocity;

  return (
    <>
      {/* Base static path */}
      <path
        ref={pathRef}
        d={pathD}
        className="stroke-gray-3"
        strokeWidth={ANIMATION_CONFIG.strokeWidthBase}
        fill="none"
        strokeLinecap="round"
      />
      {/* Animated light band traveling along path */}
      <path
        d={pathD}
        className="stroke-grayA-12"
        strokeWidth={ANIMATION_CONFIG.strokeWidthAnimated}
        fill="none"
        strokeLinecap="round"
        strokeDasharray={dashArray}
        strokeDashoffset={pathLength}
        style={{ opacity: ANIMATION_CONFIG.opacity }}
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

/**
 * Creates a straight line path between two points.
 *
 * Visual representation:
 * ```
 * from ──────────────> to
 * ```
 *
 * @param from - Starting point
 * @param to - Ending point
 * @returns SVG path string
 *
 * @example
 * ```typescript
 * straightLine({ x: 0, y: 0 }, { x: 100, y: 0 })
 * ```
 */
function straightLine(from: Point, to: Point): string {
  return renderPath([move(from), line(to)]);
}

/**
 * Creates a stepped horizontal path with three 90° angles.
 * Path goes: horizontal → vertical → horizontal
 *
 * Visual representation:
 * ```
 * from ─────┐
 *           │
 *           │
 *           └───── to
 * ```
 *
 * @param from - Starting point
 * @param to - Ending point
 * @returns SVG path string with sharp corners
 *
 * @example
 * ```typescript
 * steppedHorizontal({ x: 0, y: 0 }, { x: 100, y: 100 })
 * ```
 */
function steppedHorizontal(from: Point, to: Point): string {
  const dx = to.x - from.x;
  const midX = from.x + dx * 0.5;

  return renderPath([
    move(from),
    line({ x: midX, y: from.y }),
    line({ x: midX, y: to.y }),
    line(to),
  ]);
}

/**
 * Creates a Z-shaped path with two rounded corners.
 * Path goes: vertical → curved horizontal → vertical
 *
 * Visual representation:
 * ```
 * from
 *   │
 *   │ (vertical segment)
 *   └──╮ (first rounded corner)
 *      │ (horizontal segment)
 *   ╭──┘ (second rounded corner)
 *   │ (vertical segment)
 *   │
 *  to
 * ```
 *
 * Uses quadratic Bézier curves (Q command) for smooth 90° turns.
 * Corner radius and spacing are controlled by PATH_CONFIG constants.
 *
 * @param from - Starting point (top of Z)
 * @param to - Ending point (bottom of Z)
 * @returns SVG path string with rounded corners
 *
 * @example
 * ```typescript
 * roundedZShape({ x: 0, y: 0 }, { x: 100, y: 200 })
 *
 * // Breakdown:
 * // M 0 0          - Start at top
 * // L 0 68         - Go down to first corner
 * // Q 0 100 32 100 - Curve right (control point at 0,100)
 * // L 68 100       - Go across horizontally
 * // Q 100 100 100 132 - Curve down (control point at 100,100)
 * // L 100 200      - Go down to end
 * ```
 */
function roundedZShape(from: Point, to: Point): string {
  const dx = to.x - from.x;
  const dy = to.y - from.y;
  const direction = dx > 0 ? 1 : -1;

  const midY = from.y + dy * 0.5;
  const verticalEnd = Math.min(
    midY - PATH_CONFIG.cornerRadius,
    from.y + dy - PATH_CONFIG.minCornerSpacing - PATH_CONFIG.cornerRadius
  );

  const beforeCurve1 = { x: from.x, y: verticalEnd };
  const curve1Control = {
    x: from.x,
    y: verticalEnd + PATH_CONFIG.cornerRadius,
  };
  const afterCurve1 = {
    x: from.x + PATH_CONFIG.cornerRadius * direction,
    y: verticalEnd + PATH_CONFIG.cornerRadius,
  };

  const beforeCurve2 = {
    x: to.x - PATH_CONFIG.cornerRadius * direction,
    y: afterCurve1.y,
  };
  const curve2Control = { x: to.x, y: afterCurve1.y };
  const afterCurve2 = { x: to.x, y: to.y - PATH_CONFIG.cornerRadius };

  return renderPath([
    move(from),
    line(beforeCurve1),
    curve(curve1Control, afterCurve1),
    line(beforeCurve2),
    curve(curve2Control, afterCurve2),
    line(to),
  ]);
}
