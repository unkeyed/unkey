import type React from "react";
import {
  type PropsWithChildren,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { MAX_DRAGGABLE_WIDTH, MIN_DRAGGABLE_WIDTH } from "./constants";

const ResizablePanel = ({
  children,
  onResize,
  className,
  style,
  minW = MIN_DRAGGABLE_WIDTH,
  maxW = MAX_DRAGGABLE_WIDTH,
}: PropsWithChildren<{
  onResize?: (newWidth: number) => void;
  className: string;
  style: Record<string, unknown>;
  minW?: number;
  maxW?: number;
}>) => {
  const [isDragging, setIsDragging] = useState(false);
  const [width, setWidth] = useState<string>(String(style?.width));
  const panelRef = useRef<HTMLDivElement | null>(null);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
      e.preventDefault();
      setIsDragging(true);
    },
    []
  );

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging || !panelRef.current) {
        return;
      }

      const containerRect = panelRef.current.getBoundingClientRect();
      const newWidth = Math.min(
        Math.max(containerRect.right - e.clientX, minW),
        maxW
      );
      setWidth(`${newWidth}px`);
      onResize?.(newWidth);
    },
    [isDragging, minW, maxW, onResize]
  );

  useEffect(() => {
    if (isDragging) {
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    } else {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    }

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isDragging, handleMouseMove, handleMouseUp]);

  return (
    <div
      ref={panelRef}
      className={`relative ${className}`}
      style={{ ...style, width, right: 0, position: "fixed" }}
    >
      <div
        className="absolute top-0 left-0 w-[2px] h-full cursor-ew-resize hover:bg-primary/10 "
        onMouseDown={handleMouseDown}
      />
      {children}
    </div>
  );
};

export default ResizablePanel;
