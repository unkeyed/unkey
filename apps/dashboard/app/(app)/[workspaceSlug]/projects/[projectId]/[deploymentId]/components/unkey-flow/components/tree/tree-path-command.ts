import type { Point } from "../../types";

// SVG Path Commands:
// M (MoveTo) - Moves the pen to a point without drawing
// L (LineTo) - Draws a straight line from current point to target
// Q (Quadratic curve) - Draws a curved line using one control point

export type MoveTo = { type: "M"; point: Point };
export type LineTo = { type: "L"; point: Point };
export type QuadraticCurve = { type: "Q"; control: Point; end: Point };

export type PathCommand = MoveTo | LineTo | QuadraticCurve;

export const move = (point: Point): MoveTo => ({ type: "M", point });
export const line = (point: Point): LineTo => ({ type: "L", point });
export const curve = (control: Point, end: Point): QuadraticCurve => ({
  type: "Q",
  control,
  end,
});

/**
 * Renders a sequence of path commands into an SVG path string.
 *
 * @param commands - Array of path commands (MoveTo, LineTo, QuadraticCurve)
 * @returns SVG path data string
 * @throws {Error} If commands array is empty or doesn't start with MoveTo
 *
 * @example
 * ```typescript
 * const path = renderPath([
 *   move({ x: 0, y: 0 }),    // M 0 0 - Move pen to start
 *   line({ x: 100, y: 100 }), // L 100 100 - Draw line
 *   curve({ x: 150, y: 100 }, { x: 200, y: 150 }) // Q 150 100 200 150 - Draw curve
 * ]);
 * // Returns: "M 0 0 L 100 100 Q 150 100 200 150"
 * ```
 */
export function renderPath(commands: PathCommand[]): string {
  if (commands.length === 0) {
    throw new Error("Cannot render empty path - at least one command required");
  }

  if (commands[0].type !== "M") {
    throw new Error("Path must start with MoveTo (M) command");
  }

  return commands
    .map((cmd) => {
      switch (cmd.type) {
        case "M": // MoveTo - Move pen to point without drawing
          return `M ${cmd.point.x} ${cmd.point.y}`;
        case "L": // LineTo - Draw straight line to point
          return `L ${cmd.point.x} ${cmd.point.y}`;
        case "Q": // Quadratic curve - Draw curve using control point
          return `Q ${cmd.control.x} ${cmd.control.y} ${cmd.end.x} ${cmd.end.y}`;
        default: {
          // Exhaustive check - ensures we handle all PathCommand types
          const exhaustive: never = cmd;
          throw new Error(`Unhandled command type: ${JSON.stringify(exhaustive)}`);
        }
      }
    })
    .join(" ");
}
