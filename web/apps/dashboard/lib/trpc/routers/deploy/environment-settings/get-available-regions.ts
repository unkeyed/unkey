import { workspaceProcedure } from "../../../trpc";

export const getAvailableRegions = workspaceProcedure.query(() => {
  const regionsEnv = process.env.AVAILABLE_REGIONS ?? "eu-central-1,us-east-1";
  return regionsEnv
    .split(",")
    .map((r) => r.trim())
    .filter(Boolean);
});
