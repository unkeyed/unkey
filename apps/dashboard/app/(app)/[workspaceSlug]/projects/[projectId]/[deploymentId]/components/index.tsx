"use client";
import { useRef, useState, useCallback, useEffect } from "react";
// biome-ignore lint/style/useImportType: <explanation>
import { CanvasCustomState, InfiniteCanvasProps, Point } from "./types";

export function InfiniteCanvas({
  minZoom = 0.5,
  maxZoom = 3,
  zoomSpeed = 0.001,
  gridSize = 28,
  gridDotSize = 1.5,
  gridDotColor = "#e5e7eb",
  showGrid = true,
  onViewChange,
  children,
}: InfiniteCanvasProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [canvas, setCanvas] = useState<CanvasCustomState>({
    scale: 1,
    offset: { x: 0, y: 0 },
  });
  const [isPanning, setIsPanning] = useState(false);
  const [startPan, setStartPan] = useState<Point>({ x: 0, y: 0 });

  // Center canvas on mount
  useEffect(() => {
    const svg = svgRef.current;
    if (!svg) return;

    const rect = svg.getBoundingClientRect();
    setCanvas({
      scale: 1,
      offset: { x: rect.width / 2, y: rect.height / 2 },
    });
  }, []);

  // Notify parent of view changes
  useEffect(() => {
    onViewChange?.(canvas);
  }, [canvas, onViewChange]);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent<SVGSVGElement>) => {
      if (e.button !== 0) return;
      setIsPanning(true);
      setStartPan({
        x: e.clientX - canvas.offset.x,
        y: e.clientY - canvas.offset.y,
      });
    },
    [canvas.offset]
  );

  const handleMouseMove = useCallback(
    (e: React.MouseEvent<SVGSVGElement>) => {
      if (!isPanning) return;
      setCanvas((prev) => ({
        ...prev,
        offset: {
          x: e.clientX - startPan.x,
          y: e.clientY - startPan.y,
        },
      }));
    },
    [isPanning, startPan]
  );

  const handleMouseUp = useCallback(() => {
    setIsPanning(false);
  }, []);

  const handleWheel = useCallback(
    (e: React.WheelEvent<SVGSVGElement>) => {
      e.preventDefault();

      const svg = svgRef.current;
      if (!svg) return;

      const rect = svg.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      const delta = e.deltaY * -zoomSpeed;
      const newScale = Math.min(
        Math.max(minZoom, canvas.scale + delta),
        maxZoom
      );

      // Zoom towards mouse position
      const scaleRatio = newScale / canvas.scale;
      const newOffset = {
        x: mouseX - (mouseX - canvas.offset.x) * scaleRatio,
        y: mouseY - (mouseY - canvas.offset.y) * scaleRatio,
      };

      setCanvas({ scale: newScale, offset: newOffset });
    },
    [canvas, minZoom, maxZoom, zoomSpeed]
  );

  useEffect(() => {
    const handleGlobalMouseUp = () => setIsPanning(false);
    window.addEventListener("mouseup", handleGlobalMouseUp);
    return () => window.removeEventListener("mouseup", handleGlobalMouseUp);
  }, []);

  return (
    <svg
      ref={svgRef}
      className="w-full h-full cursor-grab active:cursor-grabbing dark:bg-black bg-gray-1"
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onWheel={handleWheel}
    >
      <g
        transform={`translate(${canvas.offset.x}, ${canvas.offset.y}) scale(${canvas.scale})`}
      >
        {showGrid && (
          <GridPattern
            size={gridSize}
            dotSize={gridDotSize}
            dotColor={gridDotColor}
          />
        )}
        {children}
      </g>
    </svg>
  );
}

function GridPattern({
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
          <circle
            cx={size / 2}
            cy={size / 2}
            r={dotSize}
            fill={dotColor}
            opacity="0.15"
          >
            <animate
              attributeName="opacity"
              values="0.15;0.25;0.15"
              dur="8s"
              repeatCount="indefinite"
              calcMode="ease"
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
