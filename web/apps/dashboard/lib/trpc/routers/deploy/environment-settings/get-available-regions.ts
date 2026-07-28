import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../../trpc";
import { listClusterRegions } from "./region-catalog";

export const getAvailableRegions = workspaceProcedure.query(async () => {
  return listClusterRegions().catch((err) => {
    console.error(err);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Unable to load regions.",
    });
  });
});
