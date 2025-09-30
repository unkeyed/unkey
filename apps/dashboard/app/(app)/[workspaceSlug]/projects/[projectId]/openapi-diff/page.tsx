"use client";
import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { ArrowRight, Magnifier } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { Loader } from "lucide-react";
import { useState } from "react";
import { Card } from "../details/card";
import { useProjectLayout } from "../layout-provider";
import { DiffViewerContent } from "./components/client";
import { DeploymentSelect } from "./deployment-select";

export default function DiffPage() {
  const { collections, isDetailsOpen } = useProjectLayout();
  const [selectedFromDeployment, setSelectedFromDeployment] = useState<string>("");
  const [selectedToDeployment, setSelectedToDeployment] = useState<string>("");

  const deployments = useLiveQuery((q) =>
    q
      .from({ deployment: collections.deployments })
      .join({ environment: collections.environments }, ({ environment, deployment }) =>
        eq(environment.id, deployment.environmentId),
      )
      .orderBy(({ deployment }) => deployment.createdAt, "desc")
      .limit(100),
  );

  const {
    data: diffData,
    isLoading: diffLoading,
    error: diffError,
  } = trpc.deploy.deployment.getOpenApiDiff.useQuery(
    {
      oldDeploymentId: selectedFromDeployment,
      newDeploymentId: selectedToDeployment,
    },
    {
      enabled: !!selectedFromDeployment && !!selectedToDeployment,
    },
  );

  const sortedDeployments = deployments.data.sort(
    (a, b) => b.deployment.createdAt - a.deployment.createdAt,
  );

  const getDeploymentLabel = (deploymentId: string): string => {
    const deployment = sortedDeployments.find((d) => d.deployment.id === deploymentId);
    if (!deployment) {
      return deploymentId;
    }

    const commitSha =
      deployment.deployment.gitCommitSha?.substring(0, 7) ||
      deployment.deployment.id.substring(0, 7);
    const branch = deployment.deployment.gitBranch || "unknown";

    return `${branch}:${commitSha}`;
  };

  const showEmptyState = !selectedFromDeployment || !selectedToDeployment;
  const showContent = selectedFromDeployment && selectedToDeployment;

  return (
    <div
      className={cn(
        "flex justify-center transition-all duration-300 ease-in-out pb-20 px-8",
        isDetailsOpen ? "w-[calc(100vw-616px)]" : "w-[calc(100vw-256px)]",
      )}
    >
      <div className="max-w-[960px] flex flex-col w-full mt-4 gap-5">
        <Card className="rounded-[14px] pt-[14px] flex flex-col overflow-hidden border-gray-4">
          {/* Header Section */}
          <div className="flex w-full justify-between items-center px-[22px]">
            <div className="flex gap-5 items-center">
              <div className="flex flex-col gap-1">
                <div className="text-accent-12 font-medium text-xs">Compare Deployments</div>
                <div className="text-gray-9 text-xs">View API changes between deployments</div>
              </div>
            </div>
          </div>

          {/* Deployment Selectors */}
          <div className="bg-gray-1 rounded-b-[14px]">
            <div className="relative h-4 flex items-center justify-center">
              <div className="absolute top-0 left-0 right-0 h-4 border-b border-gray-4 rounded-b-[14px] bg-white dark:bg-black" />
            </div>

            <div className="py-5 px-3">
              <div className="flex gap-3 items-center">
                <DeploymentSelect
                  value={selectedFromDeployment}
                  onValueChange={(value) => {
                    setSelectedFromDeployment(value);
                    if (value === selectedToDeployment) {
                      setSelectedToDeployment("");
                    }
                  }}
                  deployments={sortedDeployments}
                  isLoading={deployments.isLoading}
                  placeholder="Select baseline..."
                  disabledDeploymentId={selectedToDeployment}
                />

                <ArrowRight className="shrink-0 text-gray-9 size-[14px]" size="sm-regular" />

                <DeploymentSelect
                  value={selectedToDeployment}
                  onValueChange={(value) => {
                    setSelectedToDeployment(value);
                    if (value === selectedFromDeployment) {
                      setSelectedFromDeployment("");
                    }
                  }}
                  deployments={sortedDeployments}
                  isLoading={deployments.isLoading}
                  placeholder="Select comparison..."
                  disabledDeploymentId={selectedFromDeployment}
                />
              </div>
            </div>

            {/* Content Area - All states rendered inside card */}

            {showEmptyState && (
              <div className="flex flex-col items-center gap-4 px-8 py-12 text-center">
                {/* Icon with subtle animation */}
                <div className="relative">
                  <div className="absolute inset-0 bg-gradient-to-r from-accent-4 to-accent-3 rounded-full blur-xl opacity-20 group-hover:opacity-30 transition-opacity duration-300 animate-pulse" />
                  <div className="relative bg-gray-3 rounded-full p-3 group-hover:bg-gray-4 transition-all duration-200">
                    <Magnifier
                      className="text-gray-9 size-6 group-hover:text-gray-11 transition-all duration-200 animate-pulse"
                      style={{ animationDuration: "2s" }}
                    />
                  </div>
                </div>
                {/* Content */}
                <div className="space-y-2">
                  <h3 className="text-gray-12 font-medium text-sm">No deployments selected</h3>
                  <p className="text-gray-9 text-xs max-w-[280px] leading-relaxed">
                    Select two deployments above to compare their OpenAPI specifications and see
                    what changed between versions.
                  </p>
                </div>
              </div>
            )}

            {showContent && (
              <>
                {diffLoading && (
                  <div className="text-center py-12 mx-3 mb-3">
                    <Loader className="w-6 h-6 mx-auto mb-3 animate-spin text-gray-9" />
                    <p className="text-xs text-gray-11 font-medium">Analyzing changes...</p>
                    <p className="text-xs text-gray-9 mt-1">Comparing API specifications</p>
                  </div>
                )}

                {diffError && (
                  <div className="mx-3 mb-3 border border-error-6 rounded-md p-4 bg-error-2">
                    <div className="text-xs text-error-11 font-medium mb-1.5">
                      Failed to generate diff
                    </div>
                    <p className="text-xs text-error-11 leading-relaxed">{diffError.message}</p>
                  </div>
                )}

                {diffData && !diffLoading && (
                  <DiffViewerContent
                    changelog={diffData.changes}
                    fromDeployment={getDeploymentLabel(selectedFromDeployment)}
                    toDeployment={getDeploymentLabel(selectedToDeployment)}
                  />
                )}
              </>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}
