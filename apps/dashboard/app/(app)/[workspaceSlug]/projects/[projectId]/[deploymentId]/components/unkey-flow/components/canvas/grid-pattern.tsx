export function GridPattern({
  size,
  dotSize,
  dotColor,
}: {
  size: number;
  dotSize: number;
  dotColor: string;
}) {
  return (
    <>
      <defs>
        <pattern
          id="dot-grid"
          x={0}
          y={0}
          width={size}
          height={size}
          patternUnits="userSpaceOnUse"
        >
          <circle cx={size / 2} cy={size / 2} r={dotSize} fill={dotColor}>
            <animate
              attributeName="opacity"
              values="0.5;0.9;0.5"
              dur="3s"
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
