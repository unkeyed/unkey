"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import type { Point } from "../../types";
import { GridPattern } from "./grid-pattern";

const DEFAULT_MIN_ZOOM = 0.5;
const DEFAULT_MAX_ZOOM = 3;
const DEFAULT_ZOOM = 1;
const DEFAULT_ZOOM_SPEED = 0.001;
const DEFAULT_GRID_SIZE = 15;
const DEFAULT_DOT_RADIUS = 1.5;
const DEFAULT_DOT_CLASS = "fill-grayA-4";
const PRIMARY_MOUSE_BUTTON = 0;
const FRICTION = 0.92;
const MIN_VELOCITY = 0.5;

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
  children: React.ReactNode;
  overlay?: React.ReactNode;
};

export function InfiniteCanvas({
  minZoom = DEFAULT_MIN_ZOOM,
  maxZoom = DEFAULT_MAX_ZOOM,
  defaultZoom = DEFAULT_ZOOM,
  zoomSpeed = DEFAULT_ZOOM_SPEED,
  gridSize = DEFAULT_GRID_SIZE,
  dotRadius = DEFAULT_DOT_RADIUS,
  dotClassName = DEFAULT_DOT_CLASS,
  showGrid = true,
  children,
  overlay,
}: InfiniteCanvasProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const isPanningRef = useRef(false);
  const startPanRef = useRef<Point>({ x: 0, y: 0 });
  const velocityRef = useRef<Point>({ x: 0, y: 0 });
  const lastPosRef = useRef<Point>({ x: 0, y: 0 });
  const animationFrameRef = useRef<number | null>(null);

  const [canvas, setCanvas] = useState<CanvasState>({
    scale: defaultZoom,
    offset: { x: 0, y: 0 },
  });

  // Memoize transform string to avoid recalculation
  const transform = `translate(${canvas.offset.x},${canvas.offset.y})scale(${canvas.scale})`;

  // Initialize center offset after mount
  useEffect(() => {
    const svg = svgRef.current;
    if (!svg) {
      return;
    }

    const rect = svg.getBoundingClientRect();
    setCanvas((prev) => ({
      ...prev,
      offset: { x: rect.width / 2, y: rect.height / 4 },
    }));
  }, []);

  // Momentum animation for panning only
  const animate = useCallback(() => {
    const vx = velocityRef.current.x;
    const vy = velocityRef.current.y;

    if (Math.abs(vx) < MIN_VELOCITY && Math.abs(vy) < MIN_VELOCITY) {
      velocityRef.current = { x: 0, y: 0 };
      animationFrameRef.current = null;
      return;
    }

    velocityRef.current = {
      x: vx * FRICTION,
      y: vy * FRICTION,
    };

    setCanvas((prev) => ({
      ...prev,
      offset: {
        x: prev.offset.x + velocityRef.current.x,
        y: prev.offset.y + velocityRef.current.y,
      },
    }));

    animationFrameRef.current = requestAnimationFrame(animate);
  }, []);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent<SVGSVGElement>) => {
      if (e.button !== PRIMARY_MOUSE_BUTTON) {
        return;
      }

      if (animationFrameRef.current !== null) {
        cancelAnimationFrame(animationFrameRef.current);
        animationFrameRef.current = null;
      }

      isPanningRef.current = true;
      velocityRef.current = { x: 0, y: 0 };
      lastPosRef.current = { x: e.clientX, y: e.clientY };
      startPanRef.current = {
        x: e.clientX - canvas.offset.x,
        y: e.clientY - canvas.offset.y,
      };
    },
    [canvas.offset],
  );

  const handleMouseMove = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    if (!isPanningRef.current) {
      return;
    }

    velocityRef.current = {
      x: e.clientX - lastPosRef.current.x,
      y: e.clientY - lastPosRef.current.y,
    };

    lastPosRef.current = { x: e.clientX, y: e.clientY };

    setCanvas((prev) => ({
      ...prev,
      offset: {
        x: e.clientX - startPanRef.current.x,
        y: e.clientY - startPanRef.current.y,
      },
    }));
  }, []);

  const handleMouseUp = useCallback(() => {
    if (!isPanningRef.current) {
      return;
    }

    isPanningRef.current = false;

    if (
      Math.abs(velocityRef.current.x) > MIN_VELOCITY ||
      Math.abs(velocityRef.current.y) > MIN_VELOCITY
    ) {
      animationFrameRef.current = requestAnimationFrame(animate);
    }
  }, [animate]);

  const handleWheel = useCallback(
    (e: React.WheelEvent<SVGSVGElement>) => {
      e.preventDefault();
      e.stopPropagation();

      // Stop momentum on zoom
      if (animationFrameRef.current !== null) {
        cancelAnimationFrame(animationFrameRef.current);
        animationFrameRef.current = null;
      }
      velocityRef.current = { x: 0, y: 0 };

      const svg = svgRef.current;
      if (!svg) {
        return;
      }

      const rect = svg.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      setCanvas((prev) => {
        const delta = e.deltaY * -zoomSpeed;
        const newScale = Math.min(Math.max(minZoom, prev.scale + delta), maxZoom);

        if (newScale === prev.scale) {
          return prev;
        }

        const scaleRatio = newScale / prev.scale;
        const newOffset = {
          x: mouseX - (mouseX - prev.offset.x) * scaleRatio,
          y: mouseY - (mouseY - prev.offset.y) * scaleRatio,
        };

        return { scale: newScale, offset: newOffset };
      });
    },
    [minZoom, maxZoom, zoomSpeed],
  );

  // Cleanup animation frame on unmount
  useEffect(() => {
    return () => {
      if (animationFrameRef.current !== null) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, []);

  // Prevent browser's default scroll behavior
  useEffect(() => {
    const svg = svgRef.current;
    if (!svg) {
      return;
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
        <g transform={transform}>
          {showGrid && (
            <GridPattern gridSize={gridSize} dotRadius={dotRadius} dotClassName={dotClassName} />
          )}
          {children}
        </g>
      </svg>

      {overlay && <div className="absolute inset-0 pointer-events-none">{overlay}</div>}
    </div>
  );
}
