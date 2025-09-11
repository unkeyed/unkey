"use client";

import { trpc } from "@/lib/trpc/client";
import { Button, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { AlertCircle, ArrowLeft, GitCompare, Loader } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { DiffViewer } from "./components/client";

interface Props {
  params: {
    projectId: string;
    compare: string[]; // [from, to] or [from, to, additional-params]
  };
  searchParams: {
    [key: string]: string | string[] | undefined;
  };
}

export default function DiffPage({ params }: Props) {
  const router = useRouter();
  const [fromDeploymentId, toDeploymentId] = params.compare;
  const [selectedFromDeployment, setSelectedFromDeployment] = useState<string>(
    fromDeploymentId || "",
  );
  const [selectedToDeployment, setSelectedToDeployment] = useState<string>(toDeploymentId || "");

  // Fetch deployment details if needed in the future

  // Fetch all deployments for this project
  const { data: deploymentsData, isLoading: deploymentsLoading } = trpc.deployment.listByProject.useQuery(
    { projectId: params.projectId },
    { enabled: !!params.projectId },
  );

  const deployments = deploymentsData?.deployments || [];

  // Fetch the diff data
  const {
    data: diffData,
    isLoading: diffLoading,
    error: diffError,
  } = trpc.deployment.getOpenApiDiff.useQuery(
    {
      oldDeploymentId: selectedFromDeployment,
      newDeploymentId: selectedToDeployment,
    },
    {
      enabled: !!selectedFromDeployment && !!selectedToDeployment,
    },
  );

  // Helper function to create human-readable deployment labels
  interface DeploymentData {
    id: string;
    gitCommitSha?: string | null;
    gitBranch?: string | null;
    environment: string;
    createdAt: number;
    status: string;
  }

  const getDeploymentLabel = (deployment: DeploymentData) => {
    const commitSha = deployment.gitCommitSha?.substring(0, 7) || deployment.id.substring(0, 7);
    const branch = deployment.gitBranch || "unknown";
    const environment = deployment.environment;
    const date = new Date(deployment.createdAt).toLocaleDateString();

    return {
      primary: `${branch}:${commitSha}`,
      secondary: `${environment} • ${date}`,
      branch,
      commitSha,
      environment,
      status: deployment.status,
    };
  };

  const sortedDeployments = deployments.sort((a, b) => b.createdAt - a.createdAt);

  const handleCompare = () => {
    if (selectedFromDeployment && selectedToDeployment) {
      router.push(
        `/projects/${params.projectId}/diff/${selectedFromDeployment}/${selectedToDeployment}`,
      );
    }
  };

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-4 mb-4">
            <Button
              variant="ghost"
              onClick={() => router.push(`/projects/${params.projectId}`)}
              className="flex items-center gap-2"
            >
              <ArrowLeft className="w-4 h-4" />
              Back to Project
            </Button>
          </div>

          <div className="flex items-center gap-3">
            <div className="p-3 bg-blue-100 rounded-lg">
              <GitCompare className="w-8 h-8 text-blue-600" />
            </div>
            <div>
              <h1 className="text-3xl font-bold text-content">OpenAPI Diff</h1>
              <p className="text-content-subtle mt-1">
                Compare OpenAPI specifications between deployments
              </p>
            </div>
          </div>
        </div>

        {/* Selector Panel */}
        <div className="bg-white rounded-lg border border-border p-6 mb-8">
          <h2 className="text-xl font-semibold text-content mb-4">Select Deployments to Compare</h2>
          <p className="text-content-subtle mb-6">
            Select environments and deployments to compare their OpenAPI specifications
          </p>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {/* From Selection */}
            <div className="space-y-4">
              <div className="bg-green-50 border border-green-200 rounded-lg p-4">
                <h3 className="text-lg font-medium text-green-800 mb-4">From (Baseline)</h3>

                {/* Deployment Selection */}
                <div className="space-y-3">
                  <div>
                    <label
                      htmlFor="from-deployment-select"
                      className="block text-sm font-medium text-content mb-2"
                    >
                      Select Deployment
                    </label>
                    <Select
                      value={selectedFromDeployment}
                      onValueChange={setSelectedFromDeployment}
                      disabled={deploymentsLoading}
                    >
                      <SelectTrigger className="h-auto">
                        <SelectValue placeholder="Select baseline deployment" />
                      </SelectTrigger>
                      <SelectContent>
                        {deploymentsLoading ? (
                          <SelectItem value="loading" disabled>
                            <Loader className="w-4 h-4 mr-2 animate-spin" />
                            Loading deployments...
                          </SelectItem>
                        ) : deployments.length === 0 ? (
                          <SelectItem value="no-deployments" disabled>
                            No deployments found
                          </SelectItem>
                        ) : (
                          sortedDeployments.map((deployment) => {
                            const label = getDeploymentLabel(deployment);
                            return (
                              <SelectItem
                                key={deployment.id}
                                value={deployment.id}
                                className="h-auto py-3"
                              >
                                <div className="flex items-start justify-between w-full">
                                  <div className="flex-1">
                                    <div className="flex items-center gap-2 mb-1">
                                      <span className="font-mono text-sm font-medium">
                                        {label.primary}
                                      </span>
                                      <span
                                        className={`text-xs px-2 py-0.5 rounded-full ${label.environment === "production"
                                          ? "bg-green-100 text-green-700"
                                          : "bg-blue-100 text-blue-700"
                                          }`}
                                      >
                                        {label.environment}
                                      </span>
                                    </div>
                                    <div className="text-xs text-content-subtle">
                                      {label.secondary}
                                    </div>
                                  </div>
                                </div>
                              </SelectItem>
                            );
                          })
                        )}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>
            </div>

            {/* To Selection */}
            <div className="space-y-4">
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <h3 className="text-lg font-medium text-blue-800 mb-4">To (Comparison)</h3>

                {/* Deployment Selection */}
                <div className="space-y-3">
                  <div>
                    <label
                      htmlFor="to-deployment-select"
                      className="block text-sm font-medium text-content mb-2"
                    >
                      Select Deployment
                    </label>
                    <Select
                      value={selectedToDeployment}
                      onValueChange={setSelectedToDeployment}
                      disabled={deploymentsLoading}
                    >
                      <SelectTrigger className="h-auto">
                        <SelectValue placeholder="Select comparison deployment" />
                      </SelectTrigger>
                      <SelectContent>
                        {deploymentsLoading ? (
                          <SelectItem value="loading" disabled>
                            <Loader className="w-4 h-4 mr-2 animate-spin" />
                            Loading deployments...
                          </SelectItem>
                        ) : deployments.length === 0 ? (
                          <SelectItem value="no-deployments" disabled>
                            No deployments found
                          </SelectItem>
                        ) : (
                          sortedDeployments.map((deployment) => {
                            const label = getDeploymentLabel(deployment);
                            return (
                              <SelectItem
                                key={deployment.id}
                                value={deployment.id}
                                className="h-auto py-3"
                              >
                                <div className="flex items-start justify-between w-full">
                                  <div className="flex-1">
                                    <div className="flex items-center gap-2 mb-1">
                                      <span className="font-mono text-sm font-medium">
                                        {label.primary}
                                      </span>
                                      <span
                                        className={`text-xs px-2 py-0.5 rounded-full ${label.environment === "production"
                                          ? "bg-green-100 text-green-700"
                                          : "bg-blue-100 text-blue-700"
                                          }`}
                                      >
                                        {label.environment}
                                      </span>
                                    </div>
                                    <div className="text-xs text-content-subtle">
                                      {label.secondary}
                                    </div>
                                  </div>
                                </div>
                              </SelectItem>
                            );
                          })
                        )}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Compare Button */}
          <div className="mt-6 flex justify-center">
            <Button
              onClick={handleCompare}
              disabled={!selectedFromDeployment || !selectedToDeployment}
              className="flex items-center gap-2"
            >
              <GitCompare className="w-4 h-4" />
              Compare Deployments
            </Button>
          </div>
        </div>

        {/* Diff Results */}
        {diffLoading && (
          <div className="bg-white rounded-lg border border-border p-8 text-center">
            <Loader className="w-8 h-8 mx-auto mb-4 animate-spin text-brand" />
            <p className="text-content-subtle">Generating diff...</p>
          </div>
        )}

        {diffError && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-6">
            <div className="flex items-center gap-3">
              <AlertCircle className="w-5 h-5 text-red-600" />
              <div>
                <h3 className="font-medium text-red-800">Error generating diff</h3>
                <p className="text-red-700 mt-1">{diffError.message}</p>
              </div>
            </div>
          </div>
        )}

        {diffData && (
          <div className="bg-white rounded-lg border border-border overflow-hidden">
            <div className="border-b border-border p-6">
              <h3 className="text-lg font-semibold text-content">Diff Results</h3>
              <p className="text-content-subtle mt-1">
                {diffData.diff.changes?.length || 0} changes detected
              </p>
            </div>
            <div className="p-6">
              <DiffViewer diffData={diffData.diff} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
