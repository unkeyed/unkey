import { t } from "../../trpc";
import { createProject } from "./create";
import { listProjects } from "./list";

export const projectRouter = t.router({
  list: listProjects,
  create: createProject,
});
