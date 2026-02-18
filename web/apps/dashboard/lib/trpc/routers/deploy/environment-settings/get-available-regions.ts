import { workspaceProcedure } from "../../../trpc";

export const getAvailableRegions = workspaceProcedure.query(() => {
  const regionsEnv = process.env.AVAILABLE_REGIONS ?? "";
  return regionsEnv
    .split(",")
    .map((r) => r.trim())
    .filter(Boolean);
});
