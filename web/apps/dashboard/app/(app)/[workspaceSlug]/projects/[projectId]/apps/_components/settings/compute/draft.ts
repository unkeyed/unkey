import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { z } from "zod";

export const computeSchema = z.object({
  cpuMillicores: z.number().int().positive(),
  memoryMib: z.number().int().positive(),
  storageMib: z.number().int().min(0),
  replicas: z
    .object({ min: z.number().int().min(1), max: z.number().int().min(1) })
    .refine(({ min, max }) => min <= max, "Minimum can't be above maximum"),
  regions: z.array(z.string()).min(1, "Your app needs at least one region"),
});

export type ComputeValues = z.infer<typeof computeSchema>;

export type Replicas = ComputeValues["replicas"];

export function formatReplicas({ min, max }: Replicas): string {
  return min === max ? `${max}` : `${min}–${max}`;
}

export function readCompute(settings: EnvironmentSettings): ComputeValues {
  const first = settings.regions.at(0);
  return {
    cpuMillicores: settings.cpuMillicores,
    memoryMib: settings.memoryMib,
    storageMib: settings.storageMib,
    replicas: { min: first?.replicasMin ?? 1, max: first?.replicasMax ?? 1 },
    regions: settings.regions.map((r) => r.name),
  };
}

export function writeCompute(target: EnvironmentSettings, values: ComputeValues): void {
  target.cpuMillicores = values.cpuMillicores;
  target.memoryMib = values.memoryMib;
  target.storageMib = values.storageMib;
  target.regions = values.regions.map((name) => ({
    name,
    replicasMin: values.replicas.min,
    replicasMax: values.replicas.max,
  }));
}
