import { createContext, useContext } from "react";

type ProjectLayoutContextType = {
  isDetailsOpen: boolean;
  setIsDetailsOpen: (open: boolean) => void;
};

export const ProjectLayoutContext = createContext<ProjectLayoutContextType | null>(null);

export const useProjectLayout = () => {
  const context = useContext(ProjectLayoutContext);
  if (!context) {
    throw new Error("useProjectLayout must be used within ProjectLayout");
  }
  return context;
};
