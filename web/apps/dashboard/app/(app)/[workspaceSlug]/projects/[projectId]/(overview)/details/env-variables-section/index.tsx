"use client";

import { cn } from "@/lib/utils";
import { BracketsCurly, ChevronDown, Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { useProjectData } from "../../data-provider";
import { EmptySection } from "../domain-row";
import { AddEnvVars } from "./add-env-vars";
import { EnvVarRow } from "./env-var-row";
import { useEnvVarsManager } from "./hooks/use-env-var-manager";
import type { Environment } from "./types";

type EnvironmentVariablesSectionProps = {
  icon: ReactNode;
  title: string;
  environment: Environment;
};

const ANIMATION_CONFIG = {
  baseDelay: 200,
  itemStagger: 50,
  contentDelay: 150,
} as const;

export function EnvironmentVariablesSection({
  icon,
  environment,
  title,
}: EnvironmentVariablesSectionProps) {
  const { projectId } = useProjectData();
  const { environmentId, envVars, getExistingEnvVar, invalidate } = useEnvVarsManager({
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

  return (
    <div className="border border-gray-4 border-t-0 first:border-t first:rounded-t-[14px] last:rounded-b-[14px] w-full overflow-hidden">
      {/* Header */}
      <div
        className={cn(
          "px-4 py-3 flex justify-between items-center",
          isExpanded && "pb-0.5 pt-[14px]",
        )}
      >
        <div className="flex items-center">
          {icon}
          <div className="text-gray-12 font-medium text-xs ml-3 mr-2 capitalize">
            {title} {envVars.length > 0 && `(${envVars.length})`}
          </div>
        </div>
        <Button
          size="icon"
          variant="ghost"
          onClick={toggleExpanded}
          className="size-7 bg-gray-3 hover:bg-gray-4 mb-0.5"
        >
          <ChevronDown
            iconSize="sm-regular"
            className={cn(
              "text-accent-12 !size-3 transition-transform duration-200",
              isExpanded && "rotate-180",
            )}
          />
        </Button>
      </div>

      {/* Expandable Content */}
      <div
        className={cn(
          "bg-gray-2 rounded-b-[14px] relative transition-all duration-300 ease-in",
          isExpanded ? "opacity-100 pb-0" : "h-0 overflow-hidden opacity-0 py-0",
        )}
      >
        {/* Concave separator */}
        <div className="relative h-4 flex items-center justify-center">
          <div className="absolute top-0 left-0 right-0 h-4 border-b border-gray-4 rounded-b-[14px] bg-white dark:bg-black" />
        </div>
        <div
          className={cn(
            "transition-all duration-300 ease-out bg-gray-2",
            isExpanded ? "translate-y-0 opacity-100" : "translate-y-2 opacity-0",
          )}
          style={{
            transitionDelay: isExpanded ? `${ANIMATION_CONFIG.contentDelay}ms` : "0ms",
          }}
        >
          <div className="flex flex-col">
            {envVars.map((envVar, index) => (
              <div key={envVar.id} {...getItemAnimationProps(index, isExpanded)}>
                <EnvVarRow
                  envVar={envVar}
                  projectId={projectId}
                  getExistingEnvVar={getExistingEnvVar}
                  onDelete={invalidate}
                  onUpdate={invalidate}
                />
              </div>
            ))}

            {isAddingNew && environmentId && (
              <AddEnvVars
                environmentId={environmentId}
                getExistingEnvVar={getExistingEnvVar}
                onCancel={cancelAdding}
                onSuccess={() => {
                  invalidate();
                  cancelAdding();
                }}
              />
            )}

            {envVars.length === 0 && !isAddingNew && (
              <EmptySection
                title="No environment variables configured"
                description="Add environment variables to configure your application's runtime settings."
                className="border-none"
                icon={
                  <BracketsCurly
                    className="text-gray-9 size-6 group-hover:text-gray-11 transition-all duration-200 animate-pulse"
                    style={{ animationDuration: "2s" }}
                  />
                }
              >
                <Button size="sm" variant="primary" onClick={startAdding} className="gap-1.5 mt-1">
                  <Plus className="!size-3" />
                  Add variable
                </Button>
              </EmptySection>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

const getItemAnimationProps = (index: number, isExpanded: boolean) => {
  const prefersReduced =
    typeof window !== "undefined" &&
    window.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches;
  const delay = prefersReduced
    ? 0
    : ANIMATION_CONFIG.baseDelay + index * ANIMATION_CONFIG.itemStagger;
  return {
    className: cn(
      "transition-all duration-150 ease-out",
      isExpanded ? "translate-x-0 opacity-100" : "translate-x-2 opacity-0",
    ),
    style: { transitionDelay: isExpanded ? `${delay}ms` : "0ms" },
  };
};
