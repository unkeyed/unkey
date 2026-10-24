import type { InstanceStatus } from "@/lib/collections/deploy/instance-status";
import { ne } from "@/lib/db";
import { deployments } from "@unkey/db/src/schema";

// A skipped row records a push the platform declined to build at all (watch
// paths didn't match, auto-deploy off). Nothing was ever deployed, so the
// browsing views leave them out; a lookup by deployment id still returns them
// so their detail page keeps working.
export function excludeSkipped() {
  return ne(deployments.status, "skipped");
}

export function mapInstanceRow(row: {
  id: string;
  regionId: string;
  regionName: string;
  regionPlatform: string;
  status: InstanceStatus;
}) {
  return {
    id: row.id,
    region: {
      id: row.regionId,
      name: row.regionName,
      platform: row.regionPlatform,
    },
    status: row.status,
  };
}
