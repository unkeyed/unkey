import { createContext, useContext } from "react";

type ProjectLayoutContextType = {
  isDetailsOpen: boolean;
  setIsDetailsOpen: (open: boolean) => void;

  projectId: string;
  liveDeploymentId?: string | null;
};

export const ProjectLayoutContext = createContext<ProjectLayoutContextType | null>(null);

export const useProject = () => {
  const context = useContext(ProjectLayoutContext);
  if (!context) {
    throw new Error("useProject must be used within ProjectLayout");
  }
  return context;
};
