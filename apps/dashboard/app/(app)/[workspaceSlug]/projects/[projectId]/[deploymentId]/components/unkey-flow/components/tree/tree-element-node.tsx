import type { PropsWithChildren } from "react";
import type { Point } from "../../layout-engine";

type TreeElementNodeProps = PropsWithChildren<{
  id: string;
  position: Point;
}>;

export function TreeElementNode({ id, position, children }: TreeElementNodeProps) {
  return (
    <foreignObject x={position.x} y={position.y} width={1} height={1} overflow="visible">
      <div
        data-node-id={id}
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
