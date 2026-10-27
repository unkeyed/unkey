import { useElapsed } from "@unkey/ui";
import type { AxisEdge } from "./axis-edges";

export function AxisEdgeLabel({ edge }: { edge: AxisEdge }) {
  return edge.kind === "ago" ? <ElapsedLabel at={edge.at} /> : edge.text;
}

function ElapsedLabel({ at }: { at: number }) {
  return useElapsed(at, "narrow");
}
