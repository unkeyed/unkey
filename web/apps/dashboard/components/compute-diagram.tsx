import { Logomark } from "@/components/logomark";
import { formatCpu, formatMemory, formatStorage } from "@/lib/utils/deployment-formatters";
import {
  IconCircleInfoOutline12,
  IconHardDriveOutline18,
  IconMicrochipOutline18,
  IconRamOutline18,
} from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import type { ReactNode } from "react";

export function EdgeNode() {
  return (
    <div className="mx-auto flex items-center gap-2 rounded-lg bg-raised px-3 py-2 shadow-sm ring-1 ring-grayA-5">
      <Logomark className="size-6 rounded-md bg-grayA-3" />
      <span className="flex items-center gap-1 text-xs font-medium text-gray-12">
        Unkey edge
        <InfoTooltip
          content="Unkey's global gateway. It receives every request to your domains, applies any policies you've set, and sends it to the nearest region with a running instance."
          position={{ side: "right" }}
          className="max-w-64"
          triggerClassName="-m-1 rounded-sm p-1 text-gray-9 hover:text-gray-11 focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-grayA-12"
        >
          <IconCircleInfoOutline12 className="size-3" aria-label="What is Unkey edge?" />
        </InfoTooltip>
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
