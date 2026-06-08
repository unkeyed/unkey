"use client";

import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { ArrowRight, Magnifier } from "@unkey/icons";
import { Loading, SettingsShell } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { Card } from "../components/card";
import { useProjectData } from "../data-provider";
import { DiffViewerContent } from "./components/client";
import { DeploymentSelect } from "./components/deployment-select";
import { useDiffDeployments } from "./hooks/use-diff-deployments";

export default function DiffPage() {
  const { project } = useProjectData();
  const currentDeploymentId = project?.currentDeploymentId;
  const searchParams = useSearchParams();

  const [selectedFromDeployment, setSelectedFromDeployment] = useState<string>("");
  const [selectedToDeployment, setSelectedToDeployment] = useState<string>("");

  const { deployments: sortedDeployments, isLoading: deploymentsLoading } = useDiffDeployments();

  // Read from URL params and pre-select deployments
  useEffect(() => {
    if (deploymentsLoading) {
      return;
    }

    const fromParam = searchParams?.get("from");
    const toParam = searchParams?.get("to");

    // If URL params exist, use them
    if (fromParam && toParam) {
      const fromExists = sortedDeployments.some((d) => d.deployment.id === fromParam);
      const toExists = sortedDeployments.some((d) => d.deployment.id === toParam);

      if (fromExists) {
        setSelectedFromDeployment(fromParam);
      }
      if (toExists) {
        setSelectedToDeployment(toParam);
      }
      return;
    }

    // Otherwise, fall back to live deployment if no params
    if (currentDeploymentId) {
      const exists = sortedDeployments.some((d) => d.deployment.id === currentDeploymentId);
      if (exists) {
        setSelectedFromDeployment(currentDeploymentId);
      }
    }
  }, [currentDeploymentId, sortedDeployments, deploymentsLoading, searchParams]);

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

  const getDeploymentLabel = useCallback(
    (deploymentId: string): string => {
      const deployment = sortedDeployments.find((d) => d.deployment.id === deploymentId);
      if (!deployment) {
        return deploymentId;
      }

      return shortenId(deploymentId);
    },
    [sortedDeployments],
  );

  const showEmptyState = !selectedFromDeployment || !selectedToDeployment;
  const showContent = selectedFromDeployment && selectedToDeployment;

  return (
    <SettingsShell>
      <div className="flex flex-col gap-2 items-center">
        <span className="font-semibold text-gray-12 leading-8 text-lg">Compare Deployments</span>
        <span className="leading-4 text-gray-11 text-[13px]">
          View API changes between deployments
        </span>
      </div>

      <Card className="flex flex-col overflow-hidden w-full">
        {/* Deployment Selectors */}
        <div className="px-4 pt-6 pb-4">
          <div className="flex gap-3">
            <div className="flex flex-col gap-1.5 flex-1">
              <span className="text-[11px] font-medium text-grayA-9">Baseline</span>
              <DeploymentSelect
                value={selectedFromDeployment}
                onValueChange={(value) => {
                  setSelectedFromDeployment(value);
                  if (value === selectedToDeployment) {
                    setSelectedToDeployment("");
                  }
                }}
                deployments={sortedDeployments}
                isLoading={deploymentsLoading}
                placeholder="Select baseline..."
                disabledDeploymentId={selectedToDeployment}
              />
            </div>

            <ArrowRight className="shrink-0 text-gray-9 size-[14px] mt-8.5" iconSize="sm-regular" />

            <div className="flex flex-col gap-1.5 flex-1">
              <span className="text-[11px] font-medium text-grayA-9">Comparison</span>
              <DeploymentSelect
                value={selectedToDeployment}
                onValueChange={(value) => {
                  setSelectedToDeployment(value);
                  if (value === selectedFromDeployment) {
                    setSelectedFromDeployment("");
                  }
                }}
                deployments={sortedDeployments}
                isLoading={deploymentsLoading}
                placeholder="Select comparison..."
                disabledDeploymentId={selectedFromDeployment}
              />
            </div>
          </div>
        </div>

        {/* Content Area */}
        <div className="bg-gray-1 rounded-b-[14px] border-t border-gray-4">
          {showEmptyState && (
            <div className="flex flex-col items-center gap-4 px-8 py-12 text-center">
              <div className="relative">
                <div className="absolute inset-0 bg-linear-to-r from-accent-4 to-accent-3 rounded-full blur-xl opacity-20 transition-opacity duration-300 animate-pulse" />
                <div className="relative bg-gray-3 rounded-full p-3 transition-all duration-200">
                  <Magnifier
                    className="text-grayA-9 size-6 transition-all duration-200 animate-pulse"
                    style={{ animationDuration: "2s" }}
                  />
                </div>
              </div>
              <div className="flex flex-col gap-2">
                <h3 className="text-grayA-12 font-medium text-sm">No deployments selected</h3>
                <p className="text-grayA-9 text-xs max-w-70 leading-relaxed">
                  Select two deployments above to compare their OpenAPI specifications and see what
                  changed between versions.
                </p>
              </div>
            </div>
          )}

          {showContent && (
            <>
              {diffLoading && (
                <div className="text-center py-12 px-8">
                  <Loading className="w-6 h-6 mx-auto mb-3 text-grayA-9" />
                  <p className="text-[13px] text-grayA-11 font-medium">Analyzing changes...</p>
                  <p className="text-xs text-grayA-9 mt-1">Comparing API specifications</p>
                </div>
              )}

              {diffError && (
                <div className="p-3 pt-0">
                  <div className="flex flex-col items-center gap-4 px-8 py-12 text-center border border-error-6 rounded-lg bg-error-2">
                    <div className="relative">
                      <div className="absolute inset-0 bg-linear-to-r from-error-4 to-error-3 rounded-full blur-xl opacity-20 transition-opacity duration-300 animate-pulse" />
                      <div className="relative bg-error-3 rounded-full p-3 transition-all duration-200">
                        <Magnifier
                          className="text-error-9 size-6 transition-all duration-200 animate-pulse"
                          style={{ animationDuration: "2s" }}
                        />
                      </div>
                    </div>
                    <div className="flex flex-col gap-2">
                      <h3 className="text-error-11 font-medium text-sm">
                        Unable to compare deployments
                      </h3>
                      <p className="text-error-11 text-xs max-w-[280px] leading-relaxed opacity-90">
                        {diffError.message}
                      </p>
                    </div>
                  </div>
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
    </SettingsShell>
  );
}
