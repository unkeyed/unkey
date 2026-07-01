import type React from "react";
import { type PropsWithChildren, useCallback, useEffect, useRef, useState } from "react";

export const MAX_DRAGGABLE_WIDTH = 800;
export const MIN_DRAGGABLE_WIDTH = 300;

export const ResizablePanel = ({
  children,
  onResize,
  onClose,
  className,
  style,
  minW = MIN_DRAGGABLE_WIDTH,
  maxW = MAX_DRAGGABLE_WIDTH,
}: PropsWithChildren<{
  onResize?: (newWidth: number) => void;
  onClose: () => void;
  className: string;
  style: Record<string, unknown>;
  minW?: number;
  maxW?: number;
}>) => {
  const [isDragging, setIsDragging] = useState(false);
  const [width, setWidth] = useState<string>(String(style?.width));
  const panelRef = useRef<HTMLDivElement>(null) as React.RefObject<HTMLDivElement>;

  // biome-ignore lint/correctness/useExhaustiveDependencies: panelRef is stable and read when the event fires
  useEffect(() => {
    const handlePointerDown = (event: MouseEvent | TouchEvent) => {
      const panel = panelRef.current;
      const target = event.target;
      if (!panel || !(target instanceof Node) || panel.contains(target)) {
        return;
      }

      onClose();
    };

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("touchstart", handlePointerDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("touchstart", handlePointerDown);
    };
  }, [onClose]);

  const handleMouseDown = useCallback((e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  // biome-ignore lint/correctness/useExhaustiveDependencies: panelRef is stable and shouldn't be in deps
  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging || !panelRef.current) {
        return;
      }

      const containerRect = panelRef.current.getBoundingClientRect();
      const newWidth = Math.min(Math.max(containerRect.right - e.clientX, minW), maxW);
      setWidth(`${newWidth}px`);
      onResize?.(newWidth);
    },
    [isDragging, minW, maxW, onResize],
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
      className={`relative border-l border-gray-4 ${className}`}
      style={{ ...style, width, right: 0, position: "fixed" }}
    >
      <div
        className="absolute top-0 left-0 w-[3px] h-full cursor-ew-resize hover:bg-gray-6 transition-all z-10"
        onMouseDown={handleMouseDown}
      />
      {children}
    </div>
  );
};
