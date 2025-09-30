"use client";

import { eq, useLiveQuery } from "@tanstack/react-db";
import { Button, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { ArrowLeft, GitBranch, GitCommit, GitCompare, Globe, Tag } from "lucide-react";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import { useProjectLayout } from "../layout-provider";

export default function DiffSelectionPage(): JSX.Element {
  const { collections } = useProjectLayout();
  const params = useParams();
  const router = useRouter();
  const projectId = params?.projectId as string;

  const [selectedFromDeployment, setSelectedFromDeployment] = useState<string>("");
  const [selectedToDeployment, setSelectedToDeployment] = useState<string>("");

  // Fetch all deployments for this project
  const deployments = useLiveQuery((q) =>
    q
      .from({ deployment: collections.deployments })
      .where(({ deployment }) => eq(deployment.projectId, params?.projectId))
      .join({ environment: collections.environments }, ({ environment, deployment }) =>
        eq(environment.id, deployment.environmentId),
      )
      .orderBy(({ deployment }) => deployment.createdAt, "desc")
      .limit(100),
  );

  const handleCompare = () => {
    if (selectedFromDeployment && selectedToDeployment) {
      router.push(
        `/${params?.workspaceSlug}/projects/${projectId}/diff/${selectedFromDeployment}/${selectedToDeployment}`,
      );
    }
  };

  const sortedDeployments = deployments.data.sort(
    (a, b) => b.deployment.createdAt - a.deployment.createdAt,
  );

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-4 mb-4">
            <Button
              variant="ghost"
              onClick={() => router.push(`/projects/${projectId}`)}
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
              <h1 className="text-3xl font-bold text-content">Compare Deployments</h1>
              <p className="text-content-subtle mt-1">
                Select two deployments to compare their OpenAPI specifications
              </p>
            </div>
          </div>
        </div>

        {/* Selection Form */}
        <div className="bg-white rounded-lg border border-border p-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {/* From Selection */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-content">From (Baseline)</h3>

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
                  disabled={deployments.isLoading || deployments.data.length === 0}
                >
                  <SelectTrigger className="h-auto">
                    <SelectValue placeholder="Select baseline deployment" />
                  </SelectTrigger>
                  <SelectContent>
                    {deployments.isLoading ? (
                      <SelectItem value="loading" disabled>
                        Loading deployments...
                      </SelectItem>
                    ) : deployments.data.length === 0 ? (
                      <SelectItem value="no-deployments" disabled>
                        No deployments found
                      </SelectItem>
                    ) : (
                      sortedDeployments.map(({ deployment, environment }) => {
                        const commitSha =
                          deployment.gitCommitSha?.substring(0, 7) || deployment.id.substring(0, 7);
                        const branch = deployment.gitBranch || "unknown";
                        const date = new Date(deployment.createdAt).toLocaleDateString();

                        return (
                          <SelectItem
                            key={deployment.id}
                            value={deployment.id}
                            className="h-auto py-3"
                          >
                            <div className="flex items-start justify-between w-full">
                              <div className="flex-1">
                                <div className="flex items-center gap-2 mb-1">
                                  <GitBranch className="w-3 h-3 text-content-subtle" />
                                  <span className="font-mono text-sm font-medium">
                                    {branch}:{commitSha}
                                  </span>
                                  <div className="flex items-center gap-1">
                                    {environment?.slug === "production" ? (
                                      <Globe className="w-3 h-3 text-green-600" />
                                    ) : (
                                      <Tag className="w-3 h-3 text-blue-600" />
                                    )}
                                    <span
                                      className={`text-xs px-2 py-0.5 rounded-full ${
                                        environment?.slug === "production"
                                          ? "bg-green-100 text-green-700"
                                          : "bg-blue-100 text-blue-700"
                                      }`}
                                    >
                                      {environment?.slug}
                                    </span>
                                  </div>
                                </div>
                                <div className="text-xs text-content-subtle flex items-center gap-2">
                                  <span>
                                    {environment?.slug} • {date}
                                  </span>
                                  <span
                                    className={`px-2 py-0.5 rounded text-xs ${
                                      deployment.status === "ready"
                                        ? "bg-green-100 text-green-700"
                                        : deployment.status === "failed"
                                          ? "bg-red-100 text-red-700"
                                          : "bg-gray-100 text-gray-700"
                                    }`}
                                  >
                                    {deployment.status}
                                  </span>
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

            {/* To Selection */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-content">To (Comparison)</h3>

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
                  disabled={deployments.isLoading || deployments.data.length === 0}
                >
                  <SelectTrigger className="h-auto">
                    <SelectValue placeholder="Select comparison deployment" />
                  </SelectTrigger>
                  <SelectContent>
                    {deployments.isLoading ? (
                      <SelectItem value="loading" disabled>
                        Loading deployments...
                      </SelectItem>
                    ) : deployments.data.length === 0 ? (
                      <SelectItem value="no-deployments" disabled>
                        No deployments found
                      </SelectItem>
                    ) : (
                      sortedDeployments.map(({ deployment, environment }) => {
                        const commitSha =
                          deployment.gitCommitSha?.substring(0, 7) || deployment.id.substring(0, 7);
                        const branch = deployment.gitBranch || "unknown";
                        const date = new Date(deployment.createdAt).toLocaleDateString();
                        return (
                          <SelectItem
                            key={deployment.id}
                            value={deployment.id}
                            className="h-auto py-3"
                          >
                            <div className="flex items-start justify-between w-full">
                              <div className="flex-1">
                                <div className="flex items-center gap-2 mb-1">
                                  <GitBranch className="w-3 h-3 text-content-subtle" />
                                  <span className="font-mono text-sm font-medium">
                                    {branch}:{commitSha}
                                  </span>
                                  <div className="flex items-center gap-1">
                                    {environment?.slug === "production" ? (
                                      <Globe className="w-3 h-3 text-green-600" />
                                    ) : (
                                      <Tag className="w-3 h-3 text-blue-600" />
                                    )}
                                    <span
                                      className={`text-xs px-2 py-0.5 rounded-full ${
                                        environment?.slug === "production"
                                          ? "bg-green-100 text-green-700"
                                          : "bg-blue-100 text-blue-700"
                                      }`}
                                    >
                                      {environment?.slug}
                                    </span>
                                  </div>
                                </div>
                                <div className="text-xs text-content-subtle flex items-center gap-2">
                                  <span>
                                    {environment?.slug} • {date}
                                  </span>
                                  <span
                                    className={`px-2 py-0.5 rounded text-xs ${
                                      deployment.status === "ready"
                                        ? "bg-green-100 text-green-700"
                                        : deployment.status === "failed"
                                          ? "bg-red-100 text-red-700"
                                          : "bg-gray-100 text-gray-700"
                                    }`}
                                  >
                                    {deployment.status}
                                  </span>
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

          {/* Compare Button */}
          <div className="mt-8 flex justify-center">
            <Button
              onClick={handleCompare}
              disabled={
                !selectedFromDeployment ||
                !selectedToDeployment ||
                selectedFromDeployment === selectedToDeployment
              }
              className="flex items-center gap-2"
            >
              <GitCompare className="w-4 h-4" />
              Compare Deployments
            </Button>
          </div>

          {/* Selected deployments preview */}
          {selectedFromDeployment && selectedToDeployment && (
            <div className="mt-6 p-4 bg-gray-50 rounded-lg">
              <h4 className="text-sm font-medium text-content mb-2">Comparison Preview</h4>
              <div className="flex items-center gap-4 text-sm">
                <div className="flex items-center gap-2">
                  <GitCommit className="w-3 h-3 text-green-600" />
                  <span>
                    From: {(() => {
                      const deployment = deployments.data.find(
                        (d) => d.deployment.id === selectedFromDeployment,
                      );
                      return deployment ? deployment.deployment.id : "Unknown";
                    })()}
                  </span>
                </div>
                <span className="text-content-subtle">→</span>
                <div className="flex items-center gap-2">
                  <GitCommit className="w-3 h-3 text-blue-600" />
                  <span>
                    To: {(() => {
                      const deployment = deployments.data.find(
                        (d) => d.deployment.id === selectedToDeployment,
                      );
                      return deployment ? deployment.deployment.id : "Unknown";
                    })()}
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
