"use client";

import { plural } from "@/lib/fmt";
import { regionInfo } from "@/lib/regions";
import { formatCpu, formatMemory } from "@/lib/utils/deployment-formatters";
import { SettingsForm, SettingsGroupCollapsible } from "@unkey/ui";
import { useEnvironmentSettings } from "../environment-provider";
import { ComputeFields } from "./compute-fields";
import { type ComputeValues, formatReplicas, readCompute } from "./draft";
import { useCompute } from "./use-compute";

function computeSummary({ regions, cpuMillicores, memoryMib, replicas }: ComputeValues): string {
  return [
    regions.length > 0 ? regions.map((name) => regionInfo(name).city).join(", ") : "No regions",
    formatCpu(cpuMillicores),
    formatMemory(memoryMib),
    replicas.min === replicas.max
      ? plural(replicas.max, "instance")
      : `${formatReplicas(replicas)} instances`,
  ].join(" · ");
}

export function ComputeSettings() {
  const compute = useCompute();
  const { settings } = useEnvironmentSettings();
  return (
    <SettingsGroupCollapsible
      title="Compute"
      description="Where your app runs, how big each instance is, and how far it scales."
      summary={computeSummary(readCompute(settings))}
    >
      <SettingsForm {...compute.formProps}>
        <ComputeFields compute={compute} className="px-5 pt-4 pb-6" />
      </SettingsForm>
    </SettingsGroupCollapsible>
  );
}
