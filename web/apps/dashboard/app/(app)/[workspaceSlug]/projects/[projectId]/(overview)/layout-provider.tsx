import type { collectionManager } from "@/lib/collections";
import { createContext, useContext } from "react";

type ProjectLayoutContextType = {
  isDetailsOpen: boolean;
  setIsDetailsOpen: (open: boolean) => void;

  projectId: string;
  liveDeploymentId?: string | null;

  collections: ReturnType<typeof collectionManager.getProjectCollections>;
};

export const ProjectLayoutContext = createContext<ProjectLayoutContextType | null>(null);

export const useProject = () => {
  const context = useContext(ProjectLayoutContext);
  if (!context) {
    throw new Error("useProject must be used within ProjectLayout");
  }
  return context;
};
