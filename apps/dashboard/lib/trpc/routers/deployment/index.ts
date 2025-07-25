import { t } from "../../trpc";
import { getById } from "./getById";
import { getOpenApiDiff } from "./getOpenApiDiff";
import { listDeployments } from "./list";
import { listByBranch } from "./listByBranch";

export const deploymentRouter = t.router({
  list: listDeployments,
  listByBranch: listByBranch,
  getById: getById,
  getOpenApiDiff: getOpenApiDiff,
});
