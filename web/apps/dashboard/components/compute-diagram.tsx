import { Logomark } from "@/components/logomark";
import { formatCpu, formatMemory, formatStorage } from "@/lib/utils/deployment-formatters";
import { IconHardDriveOutline18, IconMicrochipOutline18, IconRamOutline18 } from "@unkey/icons";
import type { ReactNode } from "react";

export function EdgeNode({ subtitle = "Routes to the nearest region" }: { subtitle?: string }) {
  return (
    <div className="mx-auto flex items-center gap-2 rounded-lg bg-raised px-3 py-2 shadow-sm ring-1 ring-grayA-5">
      <Logomark className="size-6 rounded-md bg-grayA-3" />
      <span className="flex flex-col">
        <span className="text-xs font-medium text-gray-12">Unkey edge</span>
        <span className="text-2xs text-gray-10">{subtitle}</span>
      </span>
    </div>
  );
}

type Size = { cpuMillicores: number; memoryMib: number; storageMib: number };

export function sizeSpecs({ cpuMillicores, memoryMib, storageMib }: Size) {
  return [
    ["CPU", IconMicrochipOutline18, formatCpu(cpuMillicores)],
    ["RAM", IconRamOutline18, formatMemory(memoryMib)],
    ["Disk", IconHardDriveOutline18, formatStorage(storageMib)],
  ] as const;
}

export function Spec({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <span className="flex min-w-0 items-center gap-1">
      <span className="shrink-0 text-gray-10 [&_svg]:size-3" aria-hidden="true">
        {icon}
      </span>
      <span className="sr-only">{label}: </span>
      <span className="truncate text-gray-12">{value}</span>
    </span>
  );
}
