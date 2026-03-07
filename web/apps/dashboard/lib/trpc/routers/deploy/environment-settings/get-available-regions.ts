import { db } from "@/lib/db";
import { workspaceProcedure } from "../../../trpc";

export const getAvailableRegions = workspaceProcedure.query(async () => {
  const regions = await db.query.clusterRegions.findMany({
    columns: { id: true, name: true },
  });
  return regions;
});
