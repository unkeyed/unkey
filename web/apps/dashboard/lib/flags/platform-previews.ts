import { type WorkspaceFlag, projectsNav } from "./index";

export const platformPreviews = [
  {
    flag: projectsNav,
    title: "Projects navigation",
    description: "Try the new project navigation in your workspace.",
  },
] satisfies { flag: WorkspaceFlag; title: string; description: string }[];
