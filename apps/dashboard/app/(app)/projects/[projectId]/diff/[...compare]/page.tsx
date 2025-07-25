"use client";

import { trpc } from "@/lib/trpc/client";
import { Button, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { AlertCircle, ArrowLeft, GitBranch, GitCompare, Loader } from "lucide-react";
import { useRouter } from "next/navigation";
import React, { useState } from "react";
// import { sampleDiffData } from "./constants";
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

export default function DiffPage({ params, searchParams }: Props) {
  const router = useRouter();
  const [fromDeploymentId, toDeploymentId] = params.compare;
  const [selectedFromBranch, setSelectedFromBranch] = useState<string>("");
  const [selectedFromDeployment, setSelectedFromDeployment] = useState<string>(
    fromDeploymentId || "",
  );
  const [selectedToBranch, setSelectedToBranch] = useState<string>("");
  const [selectedToDeployment, setSelectedToDeployment] = useState<string>(toDeploymentId || "");

  // Fetch all branches for this project
  const { data: branchesData, isLoading: branchesLoading } = trpc.branch.listByProject.useQuery({
    projectId: params.projectId,
  });

  // Fetch deployment details to get their branch IDs
  const { data: fromDeploymentDetails } = trpc.deployment.getById.useQuery(
    { deploymentId: fromDeploymentId },
    { enabled: !!fromDeploymentId },
  );
  const { data: toDeploymentDetails } = trpc.deployment.getById.useQuery(
    { deploymentId: toDeploymentId },
    { enabled: !!toDeploymentId },
  );

  // Fetch deployments for the selected "from" branch
  const { data: fromDeploymentsData, isLoading: fromDeploymentsLoading } =
    trpc.deployment.listByBranch.useQuery(
      { branchId: selectedFromBranch },
      { enabled: !!selectedFromBranch },
    );

  // Fetch deployments for the selected "to" branch
  const { data: toDeploymentsData, isLoading: toDeploymentsLoading } =
    trpc.deployment.listByBranch.useQuery(
      { branchId: selectedToBranch },
      { enabled: !!selectedToBranch },
    );

  // Set the branch IDs based on the deployment details from URL
  React.useEffect(() => {
    if (fromDeploymentDetails?.branchId && selectedFromBranch !== fromDeploymentDetails.branchId) {
      setSelectedFromBranch(fromDeploymentDetails.branchId);
    }
  }, [fromDeploymentDetails, selectedFromBranch]);

  React.useEffect(() => {
    if (toDeploymentDetails?.branchId && selectedToBranch !== toDeploymentDetails.branchId) {
      setSelectedToBranch(toDeploymentDetails.branchId);
    }
  }, [toDeploymentDetails, selectedToBranch]);

  const handleCompare = () => {
    if (selectedFromDeployment && selectedToDeployment) {
      router.push(
        `/projects/${params.projectId}/diff/${selectedFromDeployment}/${selectedToDeployment}`,
      );
    }
  };

  const branches = branchesData?.branches || [];
  const fromDeployments = fromDeploymentsData?.deployments || [];
  const toDeployments = toDeploymentsData?.deployments || [];

  // If no deployment IDs are provided, show the comparison interface only
  if (!fromDeploymentId || !toDeploymentId) {
    return (
      <div className="min-h-screen bg-background">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          {/* Header */}
          <div className="mb-8">
            <div className="flex items-center gap-4 mb-6">
              <Button variant="outline" size="sm" onClick={() => window.history.back()}>
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back to Project
              </Button>
            </div>

            <div className="mb-6">
              <h1 className="text-3xl font-bold text-content mb-4">Compare API Deployments</h1>
              <p className="text-content-subtle">
                Select branches and deployments to compare their OpenAPI specifications
              </p>
            </div>

            {/* Comparison Selector */}
            <div className="bg-white rounded-lg border border-border p-6">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* From Section */}
                <div className="space-y-4">
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 bg-red-500 rounded-full" />
                    <h3 className="text-sm font-medium text-content-subtle">FROM</h3>
                  </div>
                  <div className="space-y-3">
                    <div>
                      <label className="block text-sm font-medium text-content mb-2">Branch</label>
                      <Select value={selectedFromBranch} onValueChange={setSelectedFromBranch}>
                        <SelectTrigger>
                          <SelectValue placeholder="Select branch" />
                        </SelectTrigger>
                        <SelectContent>
                          {branchesLoading ? (
                            <SelectItem value="loading" disabled>
                              Loading branches...
                            </SelectItem>
                          ) : branches.length === 0 ? (
                            <SelectItem value="no-branches" disabled>
                              No branches found
                            </SelectItem>
                          ) : (
                            branches.map((branch) => (
                              <SelectItem key={branch.id} value={branch.id}>
                                <div className="flex items-center gap-2">
                                  <span>{branch.name}</span>
                                  {branch.isProduction && (
                                    <span className="px-1.5 py-0.5 text-xs bg-success/10 text-success rounded">
                                      prod
                                    </span>
                                  )}
                                </div>
                              </SelectItem>
                            ))
                          )}
                        </SelectContent>
                      </Select>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-content mb-2">
                        Deployment
                      </label>
                      <Select
                        value={selectedFromDeployment}
                        onValueChange={setSelectedFromDeployment}
                        disabled={!selectedFromBranch || fromDeploymentsLoading}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select deployment" />
                        </SelectTrigger>
                        <SelectContent>
                          {fromDeploymentsLoading ? (
                            <SelectItem value="loading" disabled>
                              Loading deployments...
                            </SelectItem>
                          ) : fromDeployments.length === 0 ? (
                            <SelectItem value="no-deployments" disabled>
                              No deployments found
                            </SelectItem>
                          ) : (
                            fromDeployments.map((deployment) => (
                              <SelectItem key={deployment.id} value={deployment.id}>
                                <div className="space-y-1">
                                  <div className="flex items-center gap-2">
                                    <span className="font-mono text-xs">
                                      {deployment.gitCommitSha?.slice(0, 7) ||
                                        deployment.id.slice(0, 8)}
                                    </span>
                                    <span
                                      className={`px-1.5 py-0.5 text-xs rounded ${
                                        deployment.status === "active"
                                          ? "bg-success/10 text-success"
                                          : deployment.status === "failed"
                                            ? "bg-alert/10 text-alert"
                                            : "bg-content-subtle/10 text-content-subtle"
                                      }`}
                                    >
                                      {deployment.status}
                                    </span>
                                  </div>
                                  <div className="text-xs text-content-subtle">
                                    {new Date(deployment.createdAt).toLocaleDateString()}
                                  </div>
                                </div>
                              </SelectItem>
                            ))
                          )}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                </div>

                {/* To Section */}
                <div className="space-y-4">
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 bg-green-500 rounded-full" />
                    <h3 className="text-sm font-medium text-content-subtle">TO</h3>
                  </div>
                  <div className="space-y-3">
                    <div>
                      <label className="block text-sm font-medium text-content mb-2">Branch</label>
                      <Select value={selectedToBranch} onValueChange={setSelectedToBranch}>
                        <SelectTrigger>
                          <SelectValue placeholder="Select branch" />
                        </SelectTrigger>
                        <SelectContent>
                          {branchesLoading ? (
                            <SelectItem value="loading" disabled>
                              Loading branches...
                            </SelectItem>
                          ) : branches.length === 0 ? (
                            <SelectItem value="no-branches" disabled>
                              No branches found
                            </SelectItem>
                          ) : (
                            branches.map((branch) => (
                              <SelectItem key={branch.id} value={branch.id}>
                                <div className="flex items-center gap-2">
                                  <span>{branch.name}</span>
                                  {branch.isProduction && (
                                    <span className="px-1.5 py-0.5 text-xs bg-success/10 text-success rounded">
                                      prod
                                    </span>
                                  )}
                                </div>
                              </SelectItem>
                            ))
                          )}
                        </SelectContent>
                      </Select>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-content mb-2">
                        Deployment
                      </label>
                      <Select
                        value={selectedToDeployment}
                        onValueChange={setSelectedToDeployment}
                        disabled={!selectedToBranch || toDeploymentsLoading}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select deployment" />
                        </SelectTrigger>
                        <SelectContent>
                          {toDeploymentsLoading ? (
                            <SelectItem value="loading" disabled>
                              Loading deployments...
                            </SelectItem>
                          ) : toDeployments.length === 0 ? (
                            <SelectItem value="no-deployments" disabled>
                              No deployments found
                            </SelectItem>
                          ) : (
                            toDeployments.map((deployment) => (
                              <SelectItem key={deployment.id} value={deployment.id}>
                                <div className="space-y-1">
                                  <div className="flex items-center gap-2">
                                    <span className="font-mono text-xs">
                                      {deployment.gitCommitSha?.slice(0, 7) ||
                                        deployment.id.slice(0, 8)}
                                    </span>
                                    <span
                                      className={`px-1.5 py-0.5 text-xs rounded ${
                                        deployment.status === "active"
                                          ? "bg-success/10 text-success"
                                          : deployment.status === "failed"
                                            ? "bg-alert/10 text-alert"
                                            : "bg-content-subtle/10 text-content-subtle"
                                      }`}
                                    >
                                      {deployment.status}
                                    </span>
                                  </div>
                                  <div className="text-xs text-content-subtle">
                                    {new Date(deployment.createdAt).toLocaleDateString()}
                                  </div>
                                </div>
                              </SelectItem>
                            ))
                          )}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                </div>
              </div>

              {/* Compare Button */}
              <div className="flex justify-end mt-6">
                <Button
                  variant="primary"
                  onClick={handleCompare}
                  disabled={!selectedFromDeployment || !selectedToDeployment}
                >
                  <GitBranch className="w-4 h-4 mr-2" />
                  Compare Deployments
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Fetch the diff data using tRPC
  const {
    data: diffResult,
    isLoading,
    error,
    isError,
  } = trpc.deployment.getOpenApiDiff.useQuery({
    oldDeploymentId: fromDeploymentId,
    newDeploymentId: toDeploymentId,
  });

  // Loading state
  if (isLoading) {
    return (
      <div className="min-h-screen bg-background">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          {/* Header skeleton */}
          <div className="mb-8">
            <div className="flex items-center gap-4 mb-4">
              <Button variant="outline" size="sm" disabled>
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back
              </Button>
            </div>
            <div className="h-8 bg-background-subtle rounded w-96 mb-4 animate-pulse" />
            <div className="h-4 bg-background-subtle rounded w-64 animate-pulse" />
          </div>

          {/* Loading content */}
          <div className="flex items-center justify-center py-16">
            <div className="text-center">
              <Loader className="w-8 h-8 text-brand animate-spin mx-auto mb-4" />
              <h3 className="text-lg font-semibold text-content mb-2">Generating API Diff</h3>
              <p className="text-content-subtle">Comparing OpenAPI specifications...</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (isError || !diffResult) {
    return (
      <div className="min-h-screen bg-background">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="mb-8">
            <Button variant="outline" size="sm" onClick={() => window.history.back()}>
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back
            </Button>
          </div>

          <div className="text-center py-16">
            <AlertCircle className="w-12 h-12 text-alert mx-auto mb-4" />
            <h2 className="text-xl font-semibold text-content mb-2">Failed to Generate Diff</h2>
            <p className="text-content-subtle mb-6 max-w-md mx-auto">
              {error?.message || "Unable to compare the selected deployments. Please try again."}
            </p>
            <div className="space-x-3">
              <Button variant="outline" onClick={() => window.history.back()}>
                Go Back
              </Button>
              <Button variant="primary" onClick={() => window.location.reload()}>
                Try Again
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Success state - render the diff
  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-4 mb-6">
            <Button variant="outline" size="sm" onClick={() => window.history.back()}>
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back to Project
            </Button>
          </div>

          {/* Main Header */}
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center space-x-3">
              <GitCompare className="w-8 h-8 text-brand" />
              <div>
                <h1 className="text-2xl font-bold text-content">OpenAPI Diff Viewer</h1>
                <p className="text-sm text-content-subtle">
                  Compare API specifications and identify breaking changes
                </p>
              </div>
            </div>

            <div className="flex items-center space-x-4">
              <div className="text-sm text-content-subtle bg-background-subtle px-3 py-2 rounded-md border border-border">
                {fromDeploymentDetails?.gitCommitSha?.slice(0, 7) ||
                  fromDeploymentId?.slice(0, 8) ||
                  "v1"}{" "}
                →{" "}
                {toDeploymentDetails?.gitCommitSha?.slice(0, 7) ||
                  toDeploymentId?.slice(0, 8) ||
                  "v2"}
              </div>
            </div>
          </div>

          {/* Comparison Selector */}
          <div className="bg-white rounded-lg border border-border p-6 mb-6">
            <h2 className="text-lg font-semibold text-content mb-4">Deployment Selector</h2>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* From Section */}
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  <div className="w-3 h-3 bg-red-500 rounded-full" />
                  <h3 className="text-sm font-medium text-content-subtle">FROM</h3>
                </div>
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-content mb-2">Branch</label>
                    <Select value={selectedFromBranch} onValueChange={setSelectedFromBranch}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select branch" />
                      </SelectTrigger>
                      <SelectContent>
                        {branchesLoading ? (
                          <SelectItem value="loading" disabled>
                            Loading branches...
                          </SelectItem>
                        ) : branches.length === 0 ? (
                          <SelectItem value="no-branches" disabled>
                            No branches found
                          </SelectItem>
                        ) : (
                          branches.map((branch) => (
                            <SelectItem key={branch.id} value={branch.id}>
                              <div className="flex items-center gap-2">
                                <span>{branch.name}</span>
                                {branch.isProduction && (
                                  <span className="px-1.5 py-0.5 text-xs bg-success/10 text-success rounded">
                                    prod
                                  </span>
                                )}
                              </div>
                            </SelectItem>
                          ))
                        )}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-content mb-2">
                      Deployment
                    </label>
                    <Select
                      value={selectedFromDeployment}
                      onValueChange={setSelectedFromDeployment}
                      disabled={!selectedFromBranch || fromDeploymentsLoading}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Select deployment" />
                      </SelectTrigger>
                      <SelectContent>
                        {fromDeploymentsLoading ? (
                          <SelectItem value="loading" disabled>
                            Loading deployments...
                          </SelectItem>
                        ) : fromDeployments.length === 0 ? (
                          <SelectItem value="no-deployments" disabled>
                            No deployments found
                          </SelectItem>
                        ) : (
                          fromDeployments.map((deployment) => (
                            <SelectItem key={deployment.id} value={deployment.id}>
                              <div className="space-y-1">
                                <div className="flex items-center gap-2">
                                  <span className="font-mono text-xs">
                                    {deployment.gitCommitSha?.slice(0, 7) ||
                                      deployment.id.slice(0, 8)}
                                  </span>
                                  <span
                                    className={`px-1.5 py-0.5 text-xs rounded ${
                                      deployment.status === "active"
                                        ? "bg-success/10 text-success"
                                        : deployment.status === "failed"
                                          ? "bg-alert/10 text-alert"
                                          : "bg-content-subtle/10 text-content-subtle"
                                    }`}
                                  >
                                    {deployment.status}
                                  </span>
                                </div>
                                <div className="text-xs text-content-subtle">
                                  {new Date(deployment.createdAt).toLocaleDateString()}
                                </div>
                              </div>
                            </SelectItem>
                          ))
                        )}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>

              {/* To Section */}
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  <div className="w-3 h-3 bg-green-500 rounded-full" />
                  <h3 className="text-sm font-medium text-content-subtle">TO</h3>
                </div>
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-content mb-2">Branch</label>
                    <Select value={selectedToBranch} onValueChange={setSelectedToBranch}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select branch" />
                      </SelectTrigger>
                      <SelectContent>
                        {branchesLoading ? (
                          <SelectItem value="loading" disabled>
                            Loading branches...
                          </SelectItem>
                        ) : branches.length === 0 ? (
                          <SelectItem value="no-branches" disabled>
                            No branches found
                          </SelectItem>
                        ) : (
                          branches.map((branch) => (
                            <SelectItem key={branch.id} value={branch.id}>
                              <div className="flex items-center gap-2">
                                <span>{branch.name}</span>
                                {branch.isProduction && (
                                  <span className="px-1.5 py-0.5 text-xs bg-success/10 text-success rounded">
                                    prod
                                  </span>
                                )}
                              </div>
                            </SelectItem>
                          ))
                        )}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-content mb-2">
                      Deployment
                    </label>
                    <Select
                      value={selectedToDeployment}
                      onValueChange={setSelectedToDeployment}
                      disabled={!selectedToBranch || toDeploymentsLoading}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Select deployment" />
                      </SelectTrigger>
                      <SelectContent>
                        {toDeploymentsLoading ? (
                          <SelectItem value="loading" disabled>
                            Loading deployments...
                          </SelectItem>
                        ) : toDeployments.length === 0 ? (
                          <SelectItem value="no-deployments" disabled>
                            No deployments found
                          </SelectItem>
                        ) : (
                          toDeployments.map((deployment) => (
                            <SelectItem key={deployment.id} value={deployment.id}>
                              <div className="space-y-1">
                                <div className="flex items-center gap-2">
                                  <span className="font-mono text-xs">
                                    {deployment.gitCommitSha?.slice(0, 7) ||
                                      deployment.id.slice(0, 8)}
                                  </span>
                                  <span
                                    className={`px-1.5 py-0.5 text-xs rounded ${
                                      deployment.status === "active"
                                        ? "bg-success/10 text-success"
                                        : deployment.status === "failed"
                                          ? "bg-alert/10 text-alert"
                                          : "bg-content-subtle/10 text-content-subtle"
                                    }`}
                                  >
                                    {deployment.status}
                                  </span>
                                </div>
                                <div className="text-xs text-content-subtle">
                                  {new Date(deployment.createdAt).toLocaleDateString()}
                                </div>
                              </div>
                            </SelectItem>
                          ))
                        )}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>
            </div>

            {/* Compare Button */}
            <div className="flex justify-end mt-6">
              <Button
                variant="primary"
                onClick={handleCompare}
                disabled={!selectedFromDeployment || !selectedToDeployment}
              >
                <GitBranch className="w-4 h-4 mr-2" />
                Compare Deployments
              </Button>
            </div>
          </div>
        </div>

        {/* Diff Viewer */}
        <div className="overflow-hidden">
          <DiffViewer
            diffData={diffResult.diff}
            fromDeployment={
              fromDeploymentDetails?.gitCommitSha?.slice(0, 7) ||
              fromDeploymentId?.slice(0, 8) ||
              "v1"
            }
            toDeployment={
              toDeploymentDetails?.gitCommitSha?.slice(0, 7) || toDeploymentId?.slice(0, 8) || "v2"
            }
          />
        </div>
      </div>
    </div>
  );
}
