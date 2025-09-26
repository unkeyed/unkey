import type { collectionManager } from "@/lib/collections";
import { createContext, useContext } from "react";

type ProjectLayoutContextType = {
  isDetailsOpen: boolean;
  setIsDetailsOpen: (open: boolean) => void;
  projectId: string;
  collections: ReturnType<typeof collectionManager.getProjectCollections>;
};

export const ProjectLayoutContext = createContext<ProjectLayoutContextType | null>(null);

export const useProjectLayout = () => {
  const context = useContext(ProjectLayoutContext);
  if (!context) {
    throw new Error("useProjectLayout must be used within ProjectLayoutWrapper");
  }
  return context;
};
