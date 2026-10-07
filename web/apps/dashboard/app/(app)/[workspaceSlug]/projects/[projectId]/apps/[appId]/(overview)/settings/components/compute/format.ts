import { sizeLabel } from "@/lib/compute/sizing";
import { formatCpu, formatMemory } from "@/lib/utils/deployment-formatters";
import { match } from "@unkey/match";
import type { CardView, Replicas } from "./draft";

export function formatReplicas(replicas: Replicas): string {
  return match(replicas)
    .with({ kind: "mixed" }, () => "Mixed")
    .with({ kind: "uniform" }, ({ min, max }) => (min === max ? `${max}` : `${min}–${max}`))
    .exhaustive();
}

export function inheritedSummary(view: CardView): string {
  const single = view.replicas.kind === "uniform" && view.replicas.max === 1;
  const noun = single ? "instance" : "instances";
  return `${formatReplicas(view.replicas)} ${noun} · ${sizeLabel(view)} · ${formatCpu(view.cpuMillicores)} · ${formatMemory(view.memoryMib)}`;
}
