export type SentinelTier = {
  id: string;
  name: string;
  cpuMillicores: number;
  memoryMib: number;
};

export const SENTINEL_TIERS: SentinelTier[] = [
  { id: "s-25", name: "S-25", cpuMillicores: 250, memoryMib: 256 },
  { id: "s-50", name: "S-50", cpuMillicores: 500, memoryMib: 512 },
  { id: "s-100", name: "S-100", cpuMillicores: 1000, memoryMib: 1024 },
  { id: "s-200", name: "S-200", cpuMillicores: 2000, memoryMib: 2048 },
];

export const SENTINEL_TIERS_BY_ID = Object.fromEntries(
  SENTINEL_TIERS.map((t) => [t.id, t]),
) as Record<string, SentinelTier>;

export const DEFAULT_TIER_ID = "s-25";
