import { type WorkspaceFlag, projectsNav } from "./index";

export const platformPreviews = [
  {
    flag: projectsNav,
    title: "Projects navigation",
  },
] satisfies { flag: WorkspaceFlag; title: string }[];
