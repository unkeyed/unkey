import { shortenId } from "@/lib/shorten-id";
import { ChevronDown, CodeBranch, CodeCommit } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";
import { InfoChip } from "../../../../../details/active-deployment-card/info-chip";
import { StatusIndicator } from "../../../../../details/active-deployment-card/status-indicator";
import { ProjectDetailsContent } from "../../../../../details/project-details-expandables/project-details-content";

type ProjectDetailsProps = {
  projectId: string;
};

export const ProjectDetails = ({ projectId }: ProjectDetailsProps) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <div className="absolute top-4 left-4 pointer-events-auto">
        <div className="p-1.5 dark:bg-black bg-white rounded-lg border border-grayA-4 flex items-center justify-between gap-2 h-8 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)] w-full">
          <div className="flex gap-2 items-center">
            <StatusIndicator withSignal />
            <div className="text-accent-12 font-medium text-xs">
              {shortenId("d_58rWLre1kDtmWJ9A")}
            </div>
            <InfoChip icon={CodeBranch}>
              <span className="text-grayA-9 text-xs truncate max-w-32">
                main
              </span>
            </InfoChip>
            <InfoChip icon={CodeCommit}>
              <span className="text-grayA-9 text-xs">921103d</span>
            </InfoChip>
            <button
              onClick={() => setIsOpen(!isOpen)}
              type="button"
              className="flex-shrink-0"
            >
              <div className="w-3 h-3 flex items-center justify-center flex-shrink-0">
                <ChevronDown
                  className={cn(
                    "text-gray-8 transition-transform origin-center",
                    isOpen ? "rotate-180" : ""
                  )}
                  iconSize="sm-bold"
                />
              </div>
            </button>
          </div>
        </div>
      </div>
      <div
        className={cn(
          "absolute top-14 left-4 rounded-xl bg-gray-1 border border-grayA-4 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)] overflow-y-auto pb-4 pointer-events-auto min-w-[360px]",
          "transition-all duration-300 ease-out",
          isOpen
            ? "opacity-100 translate-y-0"
            : "opacity-0 -translate-y-2 pointer-events-none"
        )}
      >
        <ProjectDetailsContent projectId={projectId} />
      </div>
    </>
  );
};
