"use client";

import { trpc } from "@/lib/trpc/client";
import { Button, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { ArrowLeft, GitBranch } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

interface Props {
  params: {
    projectId: string;
  };
}

export default function DiffLandingPage({ params }: Props) {
  const router = useRouter();
  const [selectedFromBranch, setSelectedFromBranch] = useState<string>("");
  const [selectedFromDeployment, setSelectedFromDeployment] = useState<string>("");
  const [selectedToBranch, setSelectedToBranch] = useState<string>("");
  const [selectedToDeployment, setSelectedToDeployment] = useState<string>("");

  // Fetch all branches for this project
  const { data: branchesData, isLoading: branchesLoading } = trpc.branch.listByProject.useQuery({
    projectId: params.projectId,
  });

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
