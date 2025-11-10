"use client";
import { useRef, useState, useCallback, useEffect } from "react";
import type { Point } from "../../types";
import { GridPattern } from "./grid-pattern";

type CanvasState = {
  scale: number;
  offset: Point;
};

type InfiniteCanvasProps = {
  minZoom?: number;
  maxZoom?: number;
  defaultZoom?: number;
  zoomSpeed?: number;
  gridSize?: number;
  dotRadius?: number;
  dotClassName?: string;
  showGrid?: boolean;
  onViewChange?: (state: CanvasState) => void;
  children: React.ReactNode;
  overlay?: React.ReactNode;
};

export function InfiniteCanvas({
  minZoom = 0.5,
  maxZoom = 3,
  defaultZoom = 0.75,
  zoomSpeed = 0.001,
  gridSize = 25,
  dotRadius = 1.5,
  dotClassName = "fill-grayA-5",
  showGrid = true,
  onViewChange,
  children,
  overlay,
}: InfiniteCanvasProps) {
  const svgRef = useRef<SVGSVGElement>(null);

  // Start at scale=1 to measure nodes at their natural size.
  // We'll apply defaultZoom after children measure their dimensions.
  const [canvas, setCanvas] = useState<CanvasState>({
    scale: 1,
    offset: { x: 0, y: 0 },
  });

  const isPanningRef = useRef(false);
  const startPanRef = useRef<Point>({ x: 0, y: 0 });

  // Apply default zoom and center canvas after mount.
  // Double requestAnimationFrame ensures children have measured before we zoom.
  useEffect(() => {
    const svg = svgRef.current;
    if (!svg) {
      throw new Error("SVG ref is null during mount");
    }

    const rect = svg.getBoundingClientRect();

    // Wait two frames: first for initial render, second for measurements
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        setCanvas({
          scale: defaultZoom,
          offset: { x: rect.width / 2, y: rect.height / 2 },
        });
      });
    });
  }, [defaultZoom]);

  // Notify parent of view changes
  const onViewChangeRef = useRef(onViewChange);
  useEffect(() => {
    onViewChangeRef.current = onViewChange;
  }, [onViewChange]);

  useEffect(() => {
    onViewChangeRef.current?.(canvas);
  }, [canvas]);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent<SVGSVGElement>) => {
      if (e.button !== 0) {
        return;
      }
      isPanningRef.current = true;
      startPanRef.current = {
        x: e.clientX - canvas.offset.x,
        y: e.clientY - canvas.offset.y,
      };
    },
    [canvas.offset]
  );

  const handleMouseMove = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    if (!isPanningRef.current) {
      return;
    }
    setCanvas((prev) => ({
      ...prev,
      offset: {
        x: e.clientX - startPanRef.current.x,
        y: e.clientY - startPanRef.current.y,
      },
    }));
  }, []);

  const handleMouseUp = useCallback(() => {
    isPanningRef.current = false;
  }, []);

  const handleWheel = useCallback(
    (e: React.WheelEvent<SVGSVGElement>) => {
      e.preventDefault();
      e.stopPropagation();

      const svg = svgRef.current;
      if (!svg) {
        throw new Error("SVG ref is null during wheel event");
      }

      const rect = svg.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      const delta = e.deltaY * -zoomSpeed;
      const newScale = Math.min(
        Math.max(minZoom, canvas.scale + delta),
        maxZoom
      );

      // Zoom towards mouse position.
      // This keeps the point under the cursor stationary while zooming.
      const scaleRatio = newScale / canvas.scale;
      const newOffset = {
        x: mouseX - (mouseX - canvas.offset.x) * scaleRatio,
        y: mouseY - (mouseY - canvas.offset.y) * scaleRatio,
      };

      setCanvas({ scale: newScale, offset: newOffset });
    },
    [canvas, minZoom, maxZoom, zoomSpeed]
  );

  // Prevent browser's default scroll behavior on wheel events
  useEffect(() => {
    const svg = svgRef.current;
    if (!svg) {
      throw new Error("SVG ref is null during wheel listener setup");
    }

    const handleWheelNative = (e: WheelEvent) => {
      e.preventDefault();
    };

    svg.addEventListener("wheel", handleWheelNative, { passive: false });
    return () => svg.removeEventListener("wheel", handleWheelNative);
  }, []);

  return (
    <div className="relative w-full h-full">
      <svg
        ref={svgRef}
        className="w-full h-full cursor-grab active:cursor-grabbing dark:bg-black bg-gray-1"
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onWheel={handleWheel}
      >
        <g
          transform={`translate(${canvas.offset.x}, ${canvas.offset.y}) scale(${canvas.scale})`}
        >
          {showGrid && (
            <GridPattern
              gridSize={gridSize}
              dotRadius={dotRadius}
              dotClassName={dotClassName}
            />
          )}
          {children}
        </g>
      </svg>

      {/* Fixed overlay that doesn't pan/zoom with canvas */}
      {overlay && (
        <div className="absolute inset-0 pointer-events-none">
          <div className="pointer-events-auto">{overlay}</div>
        </div>
      )}
    </div>
  );
}
