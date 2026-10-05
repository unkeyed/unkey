import { presetFor } from "@/app/(app)/[workspaceSlug]/projects/_components/compute/sizes";
import { formatCpuParts, formatMemoryParts } from "@/lib/utils/deployment-formatters";

type Size = { cpuMillicores: number; memoryMib: number };

const joinParts = ({ value, unit }: { value: string; unit: string }) =>
  unit ? `${value} ${unit}` : value;

export function formatSize(size: Size): { label: string; cpu: string; memory: string } {
  return {
    label: presetFor(size.cpuMillicores, size.memoryMib)?.label ?? "Custom",
    cpu: joinParts(formatCpuParts(size.cpuMillicores)),
    memory: joinParts(formatMemoryParts(size.memoryMib)),
  };
}
