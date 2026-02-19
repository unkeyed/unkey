import type { PropsWithChildren } from "react";
import type { Point } from "../../layout-engine";
import { cn } from "@unkey/ui/src/lib/utils";

type TreeElementNodeProps = PropsWithChildren<{
  id: string;
  position: Point;
  size: { width: number; height: number };
}>;

// Safari foreignObject fix — see tailwind.css ".safari-fo-fix" for full documentation.
const isSafari =
  typeof navigator !== "undefined" &&
  /Safari/.test(navigator.userAgent) &&
  /Apple Computer/.test(navigator.vendor);

export function TreeElementNode({ id, position, size, children }: TreeElementNodeProps) {
  const width = size.width * 2;
  const height = size.height * 2;

  return (
    <foreignObject
      x={position.x - width / 2}
      y={position.y - height / 2}
      width={width}
      height={height}
      overflow="visible"
    >
      <div className={cn("w-full h-full flex items-center justify-center pointer-events-none", isSafari ? "safari-fo-fix" : "")}>
        <div data-node-id={id} className="pointer-events-auto">
          {children}
        </div>
      </div>
    </foreignObject>
  );
}
