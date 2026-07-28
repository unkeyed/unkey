import { and, db, inArray, ne } from "@/lib/db";
import { clusters } from "@unkey/db/src/schema";

type RegionCatalogEntry = {
  id: string;
  name: string;
  canSchedule: boolean;
};

// listClusterRegions derives logical regions from their cluster rows. Grouping
// in application code keeps the rule explicit and the result stable when a
// region contains multiple clusters.
export async function listClusterRegions(regionIds?: string[]): Promise<RegionCatalogEntry[]> {
  const rows = await db.query.clusters.findMany({
    where: and(
      ne(clusters.platform, ""),
      ne(clusters.regionName, ""),
      regionIds ? inArray(clusters.regionId, regionIds) : undefined,
    ),
    columns: {
      regionId: true,
      regionName: true,
      state: true,
    },
  });

  const regions = new Map<string, RegionCatalogEntry>();
  for (const cluster of rows) {
    const region = regions.get(cluster.regionId);
    if (region) {
      region.canSchedule ||= cluster.state === "active";
      continue;
    }
    regions.set(cluster.regionId, {
      id: cluster.regionId,
      name: cluster.regionName,
      canSchedule: cluster.state === "active",
    });
  }
  return [...regions.values()];
}
