import { t } from "../../trpc";
import { getByName } from "./getByName";
import { listByProject } from "./listByProject";

export const branchRouter = t.router({
  getByName: getByName,
  listByProject: listByProject,
});
