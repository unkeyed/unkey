import { cn } from "@/lib/utils";
import { ChevronDown, Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { AddEnvVarRow } from "./add-env-var-row";
import { EnvVarRow } from "./env-var-row";
import { useEnvVarsManager } from "./hooks/use-env-var-manager";
import type { Environment } from "./types";

type EnvironmentVariablesSectionProps = {
  icon: ReactNode;
  title: string;
  projectId: string;
  environment: Environment;
};

const ANIMATION_CONFIG = {
  baseDelay: 200,
  itemStagger: 50,
  contentDelay: 150,
} as const;

const LAYOUT_CONFIG = {
  maxContentHeight: "max-h-64",
  maxScrollHeight: "max-h-60",
} as const;

export function EnvironmentVariablesSection({
  icon,
  projectId,
  environment,
  title,
}: EnvironmentVariablesSectionProps) {
  const { envVars, getExistingEnvVar } = useEnvVarsManager({
    projectId,
    environment,
  });
  const [isExpanded, setIsExpanded] = useState(false);
  const [isAddingNew, setIsAddingNew] = useState(false);

  const toggleExpanded = () => {
    setIsExpanded((prev) => {
      const newExpanded = !prev;
      if (!newExpanded) {
        setIsAddingNew(false);
      }
      return newExpanded;
    });
  };

  const startAdding = () => setIsAddingNew(true);
  const cancelAdding = () => setIsAddingNew(false);

  const showPlusButton = isExpanded && !isAddingNew;
  const hasContent = envVars.length > 0 || isAddingNew;

  return (
    <div className="border border-gray-4 border-t-0 first:border-t first:rounded-t-[14px] last:rounded-b-[14px] w-full overflow-hidden">
      {/* Header */}
      <div className={cn("px-4 pt-3 flex justify-between items-center", isExpanded ? "" : "pb-3")}>
        <div className="flex items-center">
          {icon}
          <div className="text-gray-12 font-medium text-xs ml-3 mr-2">
            {title} {envVars.length > 0 && `(${envVars.length})`}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="icon"
            variant="ghost"
            onClick={startAdding}
            className={cn(
              "size-7 text-gray-9 hover:text-gray-11",
              showPlusButton ? "visible" : "invisible",
            )}
          >
            <Plus className="!size-3" />
          </Button>
          <Button size="icon" variant="ghost" onClick={toggleExpanded}>
            <ChevronDown
              className={cn(
                "text-grayA-9 !size-3 transition-transform duration-200",
                isExpanded && "rotate-180",
              )}
            />
          </Button>
        </div>
      </div>

      <ConcaveSeparator isExpanded={isExpanded} />

      {/* Expandable Content */}
      <div
        className={cn(
          "bg-gray-1 relative overflow-hidden transition-all duration-400 ease-in",
          isExpanded ? `${LAYOUT_CONFIG.maxContentHeight} opacity-100` : "h-0 opacity-0 py-0",
        )}
      >
        <div
          className={cn(
            "transition-all duration-300 ease-out",
            isExpanded ? "translate-y-0 opacity-100" : "translate-y-2 opacity-0",
          )}
          style={{
            transitionDelay: isExpanded ? `${ANIMATION_CONFIG.contentDelay}ms` : "0ms",
          }}
        >
          <div className={cn(LAYOUT_CONFIG.maxScrollHeight, "overflow-y-auto")}>
            {hasContent ? (
              <div className="flex flex-col">
                {envVars.map((envVar, index) => (
                  <div key={envVar.id} {...getItemAnimationProps(index, isExpanded)}>
                    <EnvVarRow
                      envVar={envVar}
                      projectId={projectId}
                      getExistingEnvVar={getExistingEnvVar}
                    />
                  </div>
                ))}

                {isAddingNew && (
                  <div {...getItemAnimationProps(envVars.length, isExpanded)}>
                    <AddEnvVarRow
                      projectId={projectId}
                      getExistingEnvVar={getExistingEnvVar}
                      onCancel={cancelAdding}
                    />
                  </div>
                )}
              </div>
            ) : (
              <EmptyState />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

const getItemAnimationProps = (index: number, isExpanded: boolean) => ({
  className: cn(
    "transition-all duration-150 ease-out",
    isExpanded ? "translate-x-0 opacity-100" : "translate-x-2 opacity-0",
  ),
  style: {
    transitionDelay: isExpanded
      ? `${ANIMATION_CONFIG.baseDelay + index * ANIMATION_CONFIG.itemStagger}ms`
      : "0ms",
  },
});

// Concave separator component
function ConcaveSeparator({ isExpanded }: { isExpanded: boolean }) {
  return (
    <div
      className={cn(
        "bg-gray-1 rounded-b-[14px] transition-all duration-200",
        isExpanded ? "opacity-100" : "opacity-0 h-0",
      )}
    >
      <div className="relative h-3 flex items-center justify-center">
        <div className="absolute top-0 left-0 right-0 h-3 border-b border-gray-4 rounded-b-[14px] bg-white dark:bg-black" />
      </div>
    </div>
  );
}

// Empty state component
function EmptyState() {
  return (
    <div className="px-4 py-8 text-center text-gray-9 text-sm flex items-center justify-center h-full">
      No environment variables configured
    </div>
  );
}
