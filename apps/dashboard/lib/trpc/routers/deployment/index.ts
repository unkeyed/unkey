import { t } from "../../trpc";
import { getById } from "./getById";
import { getOpenApiDiff } from "./getOpenApiDiff";
import { listByBranch } from "./listByBranch";
import { listDeployments } from "./list";

export const deploymentRouter = t.router({
  list: listDeployments,
  listByBranch: listByBranch,
  getById: getById,
  getOpenApiDiff: getOpenApiDiff,
});
