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
        className={className}
        style={{
          transform: "translate(-50%, -50%)",
          ...style,
        }}
      >
        {children}
      </div>
    </foreignObject>
  );
}
