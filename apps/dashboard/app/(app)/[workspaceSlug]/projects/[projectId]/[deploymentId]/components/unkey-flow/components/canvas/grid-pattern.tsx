type GridPatternProps = {
  gridSize: number; // Space between dots
  dotRadius: number; // Size of each dot
  dotClassName?: string; // Tailwind classes for dot color
};

export function GridPattern({
  gridSize,
  dotRadius,
  dotClassName,
}: GridPatternProps) {
  const animationDuration = 4;
  const maxRadiusMultiplier = 1.5;
  const minOpacity = 0.5;
  const maxOpacity = 0.8;
  const randomDelayMax = 2;

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
                dotRadius * maxRadiusMultiplier
              };${dotRadius}`}
              dur={`${animationDuration}s`}
              begin={`${Math.random() * randomDelayMax}s`}
              repeatCount="indefinite"
            />
            <animate
              attributeName="opacity"
              values={`${minOpacity};${maxOpacity};${minOpacity}`}
              dur={`${animationDuration}s`}
              begin={`${Math.random() * randomDelayMax}s`}
              repeatCount="indefinite"
            />
          </circle>
        </pattern>
      </defs>
      <rect
        x={-10000}
        y={-10000}
        width={20000}
        height={20000}
        fill="url(#dot-grid)"
      />
    </>
  );
}
