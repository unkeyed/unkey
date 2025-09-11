import { cn } from "@/lib/utils";
import { ChevronDown, Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { EnvVarRow } from "./env-var-row";
import { AddEnvVarRow } from "./add-env-var-row";
import type { Environment } from "./types";
import { useEnvVarsManager } from "./hooks/use-env-var-manager";

type EnvironmentVariablesSectionProps = {
  icon: ReactNode;
  title: string;
  projectId: string;
  environment: Environment;
};

const ANIMATION_STYLES = {
  expand: "transition-all duration-400 ease-in",
  slideIn: "transition-all duration-300 ease-out",
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
    setIsExpanded(!isExpanded);
    if (!isExpanded) {
      // Cancel adding when collapsing
      setIsAddingNew(false);
    }
  };

  const startAdding = () => {
    setIsAddingNew(true);
  };

  const cancelAdding = () => {
    setIsAddingNew(false);
  };

  const showPlusButton = isExpanded && !isAddingNew;

  return (
    <div className="border border-gray-4 border-t-0 first:border-t first:rounded-t-[14px] last:rounded-b-[14px] w-full overflow-hidden">
      {/* Header */}
      <div
        className={cn(
          "px-4 pt-3 flex justify-between items-center",
          isExpanded ? "" : "pb-3"
        )}
      >
        <div className="flex items-center">
          {icon}
          <div className="text-gray-12 font-medium text-xs ml-3 mr-2">
            {title} {envVars.length > 0 ? `(${envVars.length})` : null}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="icon"
            variant="ghost"
            onClick={startAdding}
            className={cn(
              "size-7 text-gray-9 hover:text-gray-11",
              showPlusButton ? "visible" : "invisible"
            )}
          >
            <Plus className="!size-3" />
          </Button>
          <Button size="icon" variant="ghost" onClick={toggleExpanded}>
            <ChevronDown
              className={cn(
                "text-grayA-9 !size-3 transition-transform duration-200",
                isExpanded && "rotate-180"
              )}
            />
          </Button>
        </div>
      </div>

      {/* Concave Separator */}
      <div
        className={cn(
          "bg-gray-1 rounded-b-[14px] transition-all duration-200",
          isExpanded ? "opacity-100" : "opacity-0 h-0"
        )}
      >
        <div className="relative h-3 flex items-center justify-center">
          <div className="absolute top-0 left-0 right-0 h-3 border-b border-gray-4 rounded-b-[14px] bg-white dark:bg-black" />
        </div>

        {/* Expandable Content */}
        <div
          className={cn(
            "bg-gray-1 relative overflow-hidden",
            ANIMATION_STYLES.expand,
            isExpanded ? "max-h-64 opacity-100" : "h-0 opacity-0 py-0"
          )}
        >
          <div
            className={cn(
              ANIMATION_STYLES.slideIn,
              isExpanded
                ? "translate-y-0 opacity-100"
                : "translate-y-2 opacity-0"
            )}
            style={{
              transitionDelay: isExpanded ? "150ms" : "0ms",
            }}
          >
            <div className="max-h-60 overflow-y-auto">
              {envVars.length === 0 && !isAddingNew ? (
                <div className="px-4 py-8 text-center text-gray-9 text-sm flex items-center justify-center h-full">
                  No environment variables configured
                </div>
              ) : (
                <div className="flex flex-col">
                  {envVars.map((envVar, index) => (
                    <div
                      key={envVar.id}
                      className={cn(
                        "transition-all duration-150 ease-out",
                        isExpanded
                          ? "translate-x-0 opacity-100"
                          : "translate-x-2 opacity-0"
                      )}
                      style={{
                        transitionDelay: isExpanded
                          ? `${200 + index * 50}ms`
                          : "0ms",
                      }}
                    >
                      <EnvVarRow
                        envVar={envVar}
                        projectId={projectId}
                        getExistingEnvVar={getExistingEnvVar}
                      />
                    </div>
                  ))}

                  {isAddingNew && (
                    <div
                      className={cn(
                        "transition-all duration-300 ease-out",
                        isExpanded
                          ? "translate-x-0 opacity-100"
                          : "translate-x-2 opacity-0"
                      )}
                      style={{
                        transitionDelay: isExpanded
                          ? `${200 + envVars.length * 50}ms`
                          : "0ms",
                      }}
                    >
                      <AddEnvVarRow
                        projectId={projectId}
                        getExistingEnvVar={getExistingEnvVar}
                        onCancel={cancelAdding}
                      />
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
