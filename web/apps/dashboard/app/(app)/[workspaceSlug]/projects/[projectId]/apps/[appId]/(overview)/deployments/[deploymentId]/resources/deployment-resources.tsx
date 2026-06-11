"use client";

import { MetadataCell } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/active-deployment-card/components/metadata-cell";
import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/region-flag";
import {
  formatCpuParts,
  formatMemoryParts,
  formatStorageParts,
} from "@/lib/utils/deployment-formatters";
import { InfoTooltip } from "@unkey/ui";
import { useDeployment } from "../layout-provider";

export function DeploymentResources() {
  const { deployment } = useDeployment();

  const cpu = formatCpuParts(deployment.cpuMillicores);
  const mem = formatMemoryParts(deployment.memoryMib);
  const storage = deployment.storageMib > 0 ? formatStorageParts(deployment.storageMib) : null;

  const instances = deployment.instances ?? [];
  const runningCount = instances.filter((i) => i.status === "running").length;
  const targetCount = deployment.desiredInstanceCount;

  // Prefer live instances; fall back to the desired regions before any
  // instance has reported in. Both shapes expose `region` + `flagCode`.
  const regions =
    instances.length > 0
      ? [...new Map(instances.map((i) => [i.region.id, i])).values()]
      : deployment.desiredRegions;

  return (
    <div className="grid grid-cols-2 gap-x-8 gap-y-6 md:grid-cols-3">
      <MetadataCell label="CPU">
        <span className="text-xs">
          <span className="font-medium text-gray-12">{cpu.value}</span>{" "}
          <span className="text-gray-11">{cpu.unit}</span>
        </span>
      </MetadataCell>

      <MetadataCell label="Memory">
        <span className="text-xs">
          <span className="font-medium text-gray-12">{mem.value}</span>{" "}
          <span className="text-gray-11">{mem.unit}</span>
        </span>
      </MetadataCell>

      {storage && (
        <MetadataCell label="Storage">
          <span className="text-xs">
            <span className="font-medium text-gray-12">{storage.value}</span>{" "}
            <span className="text-gray-11">{storage.unit} Disk</span>
          </span>
        </MetadataCell>
      )}

      <MetadataCell label="Instances">
        <span className="font-medium text-gray-12 text-xs">{`${runningCount} of ${targetCount}`}</span>
      </MetadataCell>

      <MetadataCell label="Regions">
        <div className="flex items-center gap-1.5">
          {regions.length > 0 ? (
            regions.map((instance) => (
              <InfoTooltip
                key={instance.region.id}
                content={instance.region.name}
                variant="inverted"
                position={{ side: "top", align: "center" }}
              >
                <RegionFlag flagCode={instance.flagCode} size="xs" shape="rounded" />
              </InfoTooltip>
            ))
          ) : (
            <span className="text-gray-11 text-xs">—</span>
          )}
        </div>
      </MetadataCell>
    </div>
  );
}
