import { t } from "../../trpc";
import { getById } from "./getById";
import { getOpenApiDiff } from "./getOpenApiDiff";
import { listDeployments } from "./list";
import { listByEnvironment } from "./listByEnvironment";
import { listByProject } from "./listByProject";

export const deploymentRouter = t.router({
  list: listDeployments,
  listByEnvironment: listByEnvironment,
  listByProject: listByProject,
  getById: getById,
  getOpenApiDiff: getOpenApiDiff,
});
