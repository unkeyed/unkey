import { db } from "@/lib/db";
import { workspaceProcedure } from "../../../trpc";

export const getAvailableRegions = workspaceProcedure.query(async () => {
  const r = await db.query.regions.findMany({
    columns: { id: true, name: true, canSchedule: true },
  });
  return r;
});
