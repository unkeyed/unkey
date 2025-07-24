import { t } from "../../trpc";
import { getById } from "./getById";
import { getOpenApiDiff } from "./getOpenApiDiff";
import { listByBranch } from "./listByBranch";
import { listVersions } from "./list";

export const versionRouter = t.router({
  list: listVersions,
  listByBranch: listByBranch,
  getById: getById,
  getOpenApiDiff: getOpenApiDiff,
});
