import { t } from "@/lib/trpc/trpc";
import { getById } from "./getById";
import { getOpenApiDiff } from "./getOpenApiDiff";
import { listDeployments } from "./list";

export const deploymentRouter = t.router({
  list: listDeployments,
  getById: getById,
  getOpenApiDiff: getOpenApiDiff,
});
