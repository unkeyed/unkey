import { createContext, useContext } from "react";

type ProjectLayoutContextType = {
  isDetailsOpen: boolean;
  setIsDetailsOpen: (open: boolean) => void;

  // Active deployment ID for the production environment.
  // Must be fetched on the project list screen and passed down to this component.
  // Required by ActiveDeploymentCard and ProjectDetailsExpandable components.
  activeDeploymentId: string;
  projectId: string;
};

export const ProjectLayoutContext = createContext<ProjectLayoutContextType | null>(null);

export const useProjectLayout = () => {
  const context = useContext(ProjectLayoutContext);
  if (!context) {
    throw new Error("useProjectLayout must be used within ProjectLayoutWrapper");
  }
  return context;
};
