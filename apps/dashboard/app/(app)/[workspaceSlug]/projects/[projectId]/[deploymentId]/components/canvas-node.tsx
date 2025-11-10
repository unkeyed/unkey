type CanvasNodeProps = {
  x: number;
  y: number;
  children: React.ReactNode;
  className?: string;
  style?: React.CSSProperties;
  onClick?: () => void;
};

export function CanvasNode({
  x,
  y,
  children,
  className = "",
  style = {},
  onClick,
}: CanvasNodeProps) {
  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
    <foreignObject
      x={x}
      y={y}
      width={1}
      height={1}
      overflow="visible"
      onClick={onClick}
      style={{ cursor: onClick ? "pointer" : "default" }}
    >
      <div
        style={{
          position: "absolute",
          left: 0,
          top: 0,
          transform: "translate(-50%, -50%)",
          ...style,
        }}
        className={className}
      >
        {children}
      </div>
    </foreignObject>
  );
}
