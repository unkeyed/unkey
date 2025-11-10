import { type PropsWithChildren, useEffect, useRef } from "react";
import type { Point } from "../../types";

/**
 * Wrapper that measures rendered node dimensions via DOM.
 * Reports dimensions back to parent on mount and when content changes.
 * @throws Error if ref is null or dimensions are zero
 */
type TreeElementNodeProps = PropsWithChildren<{
  id: string;
  position: Point;
  onMeasure: (id: string, width: number, height: number) => void;
}>;

export function TreeElementNode({
  id,
  position,
  children,
  onMeasure,
}: TreeElementNodeProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ref.current) {
      throw new Error(`Failed to get DOM ref for node ${id}`);
    }

    const { width, height } = ref.current.getBoundingClientRect();

    if (width === 0 || height === 0) {
      throw new Error(
        `Node ${id} has invalid dimensions: ${width}x${height}px`
      );
    }

    onMeasure(id, width, height);
  }, [id, onMeasure, children]);

  return (
    <foreignObject
      x={position.x}
      y={position.y}
      width={1}
      height={1}
      overflow="visible"
    >
      <div
        ref={ref}
        style={{
          position: "absolute",
          left: 0,
          top: 0,
          transform: "translate(-50%, -50%)",
        }}
      >
        {children}
      </div>
    </foreignObject>
  );
}
