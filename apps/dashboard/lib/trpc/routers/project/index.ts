import { t } from "../../trpc";
import { listProjectBranches } from "./branches";
import { createProject } from "./create";

export const projectRouter = t.router({
  create: createProject,
  branches: listProjectBranches,
});
