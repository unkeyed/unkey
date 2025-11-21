import { useMemo, useRef } from "react";
import type { Point } from "../../types";
import {
  type LineTo,
  type PathCommand,
  type QuadraticCurve,
  curve,
  line,
  move,
  renderPath,
} from "./tree-path-command";

const CORNER_RADIUS = 32;

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
  path: Point[];
  animation?: AnimationConfig;
};

/**
 * Animated connection line following a path of points.
 * Draws rounded corners at direction changes for visual clarity.
 */
export function TreeConnectionLine({
  path,
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

    const defaults = ANIMATION_PRESETS.dots;
    return {
      dashLength: animation.custom.dashLength ?? defaults.dashLength,
      gapLength: animation.custom.gapLength ?? defaults.gapLength,
      speed: animation.custom.speed ?? defaults.speed,
      strokeWidth: animation.custom.strokeWidth ?? defaults.strokeWidth,
      color: animation.custom.color ?? defaults.color,
    };
  }, [animation]);

  const pathD = useMemo(() => buildPath(path), [path]);

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
        to={`-${dashTotal}`}
        from="0"
        dur={`${animConfig.speed}s`}
        repeatCount="indefinite"
      />
    </path>
  );
}

/**
 * Convert array of points into SVG path with rounded corners.
 *
 * Handles three cases:
 * 1. Empty/single point: invalid, return empty
 * 2. Two points: straight line
 * 3. Multiple points: lines with rounded corners at direction changes
 */
function buildPath(points: Point[]): string {
  if (points.length < 2) {
    return "";
  }
  if (points.length === 2) {
    return renderPath([move(points[0]), line(points[1])]);
  }

  const commands: PathCommand[] = [move(points[0])];

  for (let i = 0; i < points.length - 1; i++) {
    const segment = {
      current: points[i],
      next: points[i + 1],
      afterNext: points[i + 2], // undefined on last segment
    };

    if (segment.afterNext === undefined) {
      // Last segment: straight line to end
      commands.push(line(segment.next));
      continue;
    }

    if (hasCorner(segment.current, segment.next, segment.afterNext)) {
      commands.push(...buildRoundedCorner(segment.current, segment.next, segment.afterNext));
    } else {
      // No direction change: straight line
      commands.push(line(segment.next));
    }
  }

  return renderPath(commands);
}

/**
 * Check if three consecutive points form a corner (direction change).
 * Returns true if path changes from horizontal to vertical or vice versa.
 */
function hasCorner(current: Point, corner: Point, next: Point): boolean {
  const toCorner = {
    dx: corner.x - current.x,
    dy: corner.y - current.y,
  };

  const fromCorner = {
    dx: next.x - corner.x,
    dy: next.y - corner.y,
  };

  // Corner exists if direction changes from H→V or V→H
  return (
    (toCorner.dx !== 0 && toCorner.dy === 0 && fromCorner.dx === 0 && fromCorner.dy !== 0) ||
    (toCorner.dx === 0 && toCorner.dy !== 0 && fromCorner.dx !== 0 && fromCorner.dy === 0)
  );
}

/**
 * Build commands for a rounded corner at the middle point.
 *
 * Creates two commands:
 * 1. Line to entry point (before corner)
 * 2. Curve through corner to exit point (after corner)
 *
 * Visual:
 * ```
 * current ────→ entry ╮
 *                     │ curve (control at corner)
 *                exit ↓
 *                     │
 *                    next
 * ```
 */
function buildRoundedCorner(current: Point, corner: Point, next: Point): [LineTo, QuadraticCurve] {
  const toCorner = {
    dx: corner.x - current.x,
    dy: corner.y - current.y,
  };

  const fromCorner = {
    dx: next.x - corner.x,
    dy: next.y - corner.y,
  };

  // Calculate entry point (before corner)
  const entryDistance = Math.sqrt(toCorner.dx ** 2 + toCorner.dy ** 2);
  const entryRadius = Math.min(CORNER_RADIUS, entryDistance / 2);
  const entryRatio = (entryDistance - entryRadius) / entryDistance;

  const entryPoint = {
    x: current.x + toCorner.dx * entryRatio,
    y: current.y + toCorner.dy * entryRatio,
  };

  // Calculate exit point (after corner)
  const exitDistance = Math.sqrt(fromCorner.dx ** 2 + fromCorner.dy ** 2);
  const exitRadius = Math.min(CORNER_RADIUS, exitDistance / 2);
  const exitRatio = exitRadius / exitDistance;

  const exitPoint = {
    x: corner.x + fromCorner.dx * exitRatio,
    y: corner.y + fromCorner.dy * exitRatio,
  };

  return [
    line(entryPoint),
    curve(corner, exitPoint), // Control point at corner creates smooth turn
  ];
}
