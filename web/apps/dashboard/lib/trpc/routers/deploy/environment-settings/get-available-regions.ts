import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../../trpc";

export const getAvailableRegions = workspaceProcedure.query(async () => {
  return db.query.regions
    .findMany({
      columns: { id: true, name: true, canSchedule: true },
    })
    .catch((err) => {
      console.error(err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Unable to load regions.",
      });
    });
});
