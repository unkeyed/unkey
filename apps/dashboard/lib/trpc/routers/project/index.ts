import { t } from "../../trpc";
import { listProjectBranches } from "./branches";
import { createProject } from "./create";
import { listProjects } from "./list";

export const projectRouter = t.router({
  list: listProjects,
  create: createProject,
  branches: listProjectBranches,
});
