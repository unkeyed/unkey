const ANIMATION_DURATION = 4;
const MAX_RADIUS_MULTIPLIER = 1.5;
const MIN_OPACITY = 0.5;
const MAX_OPACITY = 0.8;
const RANDOM_DELAY_MAX = 2;
const GRID_OFFSET = -10000;
const GRID_DIMENSION = 20000;

type GridPatternProps = {
  gridSize: number; // Space between dots
  dotRadius: number; // Size of each dot
  dotClassName?: string; // classes for dot color
};

export function GridPattern({
  gridSize,
  dotRadius,
  dotClassName,
}: GridPatternProps) {
  return (
    <>
      <defs>
        <pattern
          id="dot-grid"
          x={0}
          y={0}
          width={gridSize}
          height={gridSize}
          patternUnits="userSpaceOnUse"
        >
          <circle
            cx={gridSize / 2}
            cy={gridSize / 2}
            r={dotRadius}
            className={dotClassName}
          >
            <animate
              attributeName="r"
              values={`${dotRadius};${
                dotRadius * MAX_RADIUS_MULTIPLIER
              };${dotRadius}`}
              dur={`${ANIMATION_DURATION}s`}
              begin={`${Math.random() * RANDOM_DELAY_MAX}s`}
              repeatCount="indefinite"
            />
            <animate
              attributeName="opacity"
              values={`${MIN_OPACITY};${MAX_OPACITY};${MIN_OPACITY}`}
              dur={`${ANIMATION_DURATION}s`}
              begin={`${Math.random() * RANDOM_DELAY_MAX}s`}
              repeatCount="indefinite"
            />
          </circle>
        </pattern>
      </defs>
      <rect
        x={GRID_OFFSET}
        y={GRID_OFFSET}
        width={GRID_DIMENSION}
        height={GRID_DIMENSION}
        fill="url(#dot-grid)"
      />
    </>
  );
}
