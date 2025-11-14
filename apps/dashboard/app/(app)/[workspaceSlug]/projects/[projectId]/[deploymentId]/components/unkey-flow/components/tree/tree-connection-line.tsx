import { useRef, useMemo } from "react";
import type { Point } from "../../types";
import {
  renderPath,
  move,
  line,
  curve,
  type PathCommand,
} from "./tree-path-command";

const PATH_CONFIG = {
  straightLineThreshold: 20,
  minVerticalForCurve: 100,
  cornerRadius: 32,
  minCornerSpacing: 64,
} as const;

// Preset animation styles
const ANIMATION_PRESETS = {
  dots: {
    dashLength: 0.1,
    gapLength: 8,
    speed: 2,
    strokeWidth: 3,
    color: "stroke-gray-8",
  },
  dashes: {
    dashLength: 8,
    gapLength: 8,
    speed: 1,
    strokeWidth: 2.5,
    color: "stroke-gray-8",
  },
  "dots-slow": {
    dashLength: 0.1,
    gapLength: 12,
    speed: 8,
    strokeWidth: 3,
    color: "stroke-gray-8",
  },
  "dashes-fast": {
    dashLength: 6,
    gapLength: 6,
    speed: 0.5,
    strokeWidth: 2.5,
    color: "stroke-gray-8",
  },
  pulse: {
    dashLength: 4,
    gapLength: 16,
    speed: 3,
    strokeWidth: 3.5,
    color: "stroke-gray-8",
  },
} as const;

type AnimationPreset = keyof typeof ANIMATION_PRESETS;

type CustomAnimationConfig = {
  dashLength: number;
  gapLength: number;
  speed: number;
  strokeWidth: number;
  color: string;
};

type AnimationConfig =
  | { preset: AnimationPreset; color?: string }
  | { custom: Partial<CustomAnimationConfig> };

type TreeConnectionLineProps = {
  from: Point;
  to: Point;
  waypoints?: Point[];
  horizontal?: boolean;
  animation?: AnimationConfig;
};

/**
 * Animated connection line between two points in a tree layout.
 * Draws curved paths for visual clarity, straight lines when points are close.
 * Includes animated dashes traveling along the path.
 */
export function TreeConnectionLine({
  from,
  to,
  waypoints,
  horizontal = false,
  animation = { preset: "dots" },
}: TreeConnectionLineProps) {
  const pathRef = useRef<SVGPathElement>(null);

  const animConfig = useMemo((): CustomAnimationConfig => {
    if ("preset" in animation) {
      const preset = ANIMATION_PRESETS[animation.preset];
      return {
        ...preset,
        color: animation.color ?? preset.color,
      };
    }

    // Merge custom config with defaults
    const defaults = ANIMATION_PRESETS.dots;
    return {
      dashLength: animation.custom.dashLength ?? defaults.dashLength,
      gapLength: animation.custom.gapLength ?? defaults.gapLength,
      speed: animation.custom.speed ?? defaults.speed,
      strokeWidth: animation.custom.strokeWidth ?? defaults.strokeWidth,
      color: animation.custom.color ?? defaults.color,
    };
  }, [animation]);

  const pathD = useMemo(() => {
    // If waypoints exist, create stepped path with rounded corners
    if (waypoints && waypoints.length > 0) {
      return steppedPathWithRoundedCorners(from, waypoints, to);
    }

    if (horizontal) {
      const dy = Math.abs(to.y - from.y);
      return dy < PATH_CONFIG.cornerRadius * 2
        ? straightLine(from, to)
        : steppedHorizontal(from, to);
    }

    const dx = Math.abs(to.x - from.x);
    const dy = to.y - from.y;
    const needsStraightLine =
      dx < PATH_CONFIG.straightLineThreshold ||
      dy < PATH_CONFIG.minVerticalForCurve;

    return needsStraightLine ? straightLine(from, to) : roundedZShape(from, to);
  }, [from, to, waypoints, horizontal]);

  const dashArray = `${animConfig.dashLength} ${animConfig.gapLength}`;
  const dashTotal = animConfig.dashLength + animConfig.gapLength;

  return (
    <path
      ref={pathRef}
      d={pathD}
      className={animConfig.color}
      strokeWidth={animConfig.strokeWidth}
      fill="none"
      strokeLinecap="round"
      strokeDasharray={dashArray}
    >
      <animate
        attributeName="stroke-dashoffset"
        from="0"
        to={dashTotal.toString()}
        dur={`${animConfig.speed}s`}
        repeatCount="indefinite"
      />
    </path>
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

/**
 * Creates a stepped path through waypoints with rounded corners.
 *
 * Visual representation:
 * ```
 * from ───┐
 *         │
 *         ╰───┐
 *             │
 *             ╰──→ to
 * ```
 */
function steppedPathWithRoundedCorners(
  from: Point,
  waypoints: Point[],
  to: Point
): string {
  const radius = PATH_CONFIG.cornerRadius;
  const commands: PathCommand[] = [move(from)];

  const points = [from, ...waypoints, to];

  for (let i = 0; i < points.length - 1; i++) {
    const current = points[i];
    const next = points[i + 1];

    // Check if there's a corner at 'next'
    if (i < points.length - 2) {
      const afterNext = points[i + 2];

      // Calculate direction vectors
      const dx1 = next.x - current.x;
      const dy1 = next.y - current.y;
      const dx2 = afterNext.x - next.x;
      const dy2 = afterNext.y - next.y;

      // Detect direction change (corner)
      const isCorner =
        (dx1 !== 0 && dy1 === 0 && dx2 === 0 && dy2 !== 0) ||
        (dx1 === 0 && dy1 !== 0 && dx2 !== 0 && dy2 === 0);

      if (isCorner) {
        // Line to point before corner
        const segmentLength = Math.sqrt(dx1 * dx1 + dy1 * dy1);
        const clampedRadius = Math.min(radius, segmentLength / 2);
        const ratio = (segmentLength - clampedRadius) / segmentLength;

        const beforeCorner = {
          x: current.x + dx1 * ratio,
          y: current.y + dy1 * ratio,
        };

        commands.push(line(beforeCorner));

        // Calculate the point after the corner
        const nextSegmentLength = Math.sqrt(dx2 * dx2 + dy2 * dy2);
        const nextClampedRadius = Math.min(radius, nextSegmentLength / 2);
        const afterCornerRatio = nextClampedRadius / nextSegmentLength;

        const afterCorner = {
          x: next.x + dx2 * afterCornerRatio,
          y: next.y + dy2 * afterCornerRatio,
        };

        // Curved corner through 'next' to 'afterCorner'
        commands.push(curve(next, afterCorner));

        continue;
      }
    }

    // No corner - straight line
    commands.push(line(next));
  }

  return renderPath(commands);
}
