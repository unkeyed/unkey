import { db, isNull } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { sentinelTiers } from "@unkey/db/src/schema";

// listTiers returns currently-offered sentinel tiers, i.e. rows whose
// effective_until is NULL. Used to populate the tier picker in the UI.
export const listTiers = workspaceProcedure.use(withRatelimit(ratelimit.read)).query(async () => {
  return db
    .select({
      tierId: sentinelTiers.tierId,
      version: sentinelTiers.version,
      cpuMillicores: sentinelTiers.cpuMillicores,
      memoryMib: sentinelTiers.memoryMib,
      pricePerSecond: sentinelTiers.pricePerSecond,
    })
    .from(sentinelTiers)
    .where(isNull(sentinelTiers.effectiveUntil))
    .orderBy(sentinelTiers.cpuMillicores);
});
