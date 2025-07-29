"use client";

import { trpc } from "@/lib/trpc/client";
import { Button } from "@unkey/ui";
import {
  Activity,
  ArrowLeft,
  ChevronRight,
  Clock,
  ExternalLink,
  Eye,
  FolderOpen,
  GitCommit,
  Globe,
  MoreVertical,
  Play,
  Plus,
  RotateCcw,
  Search,
  Settings,
  Tag,
} from "lucide-react";
import { useParams } from "next/navigation";
import { useState } from "react";

// Type definitions - removed unused Project interface

export default function ProjectDetailPage(): JSX.Element {
  const params = useParams();
  const projectId = params?.projectId as string;
  const [activeTab, setActiveTab] = useState<"overview" | "deployments" | "settings">("overview");
  const [searchTerm, setSearchTerm] = useState<string>("");

  // Get project deployments
  const { data, isLoading, error } = trpc.deployment.listByProject.useQuery(
    { projectId },
    {
      enabled: !!projectId,
    },
  );

  // Handle invalid project ID
  if (!projectId) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-content mb-2">Invalid Project ID</h1>
          <p className="text-content-subtle mb-4">The project URL is malformed.</p>
          <a
            href="/projects"
            className="inline-flex items-center gap-2 px-4 py-2 border border-border rounded-lg text-content hover:bg-background-subtle transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to Projects
          </a>
        </div>
      </div>
    );
  }

  // Handle loading state
  if (isLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="flex items-center gap-3">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand" />
          <span className="text-content-subtle">Loading project...</span>
        </div>
      </div>
    );
  }

  // Handle error state
  if (error) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-alert mb-2">Error Loading Project</h1>
          <p className="text-content-subtle mb-4">Failed to load project: {error.message}</p>
          <div className="flex gap-3 justify-center">
            <a
              href="/projects"
              className="inline-flex items-center gap-2 px-4 py-2 border border-border rounded-lg text-content hover:bg-background-subtle transition-colors"
            >
              <ArrowLeft className="w-4 h-4" />
              Back to Projects
            </a>
            <Button variant="primary" onClick={() => window.location.reload()}>
              Try Again
            </Button>
          </div>
        </div>
      </div>
    );
  }

  // Handle no data
  if (!data) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-content mb-2">Project not found</h1>
          <p className="text-content-subtle mb-4">The project you're looking for doesn't exist.</p>
          <a
            href="/projects"
            className="inline-flex items-center gap-2 px-4 py-2 border border-border rounded-lg text-content hover:bg-background-subtle transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to Projects
          </a>
        </div>
      </div>
    );
  }

  // Get project and deployments from the query
  const project = data?.project || null;
  const deployments = data?.deployments || [];

  const filteredDeployments = deployments.filter(
    (deployment) =>
      deployment.gitBranch?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      deployment.gitCommitSha?.toLowerCase().includes(searchTerm.toLowerCase()),
  );

  const getStatusColor = (status: string): string => {
    switch (status) {
      case "active":
        return "text-success bg-success/10 border-success/20";
      case "deploying":
        return "text-warn bg-warn/10 border-warn/20";
      case "failed":
        return "text-alert bg-alert/10 border-alert/20";
      case "pending":
        return "text-warn bg-warn/10 border-warn/20";
      case "archived":
        return "text-content-subtle bg-background-subtle border-border";
      default:
        return "text-content-subtle bg-background-subtle border-border";
    }
  };

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-4 mb-4">
            <a
              href="/projects"
              className="inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-content-subtle hover:text-content transition-colors rounded-md hover:bg-background-subtle"
            >
              <ArrowLeft className="w-4 h-4" />
              Back to Projects
            </a>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-brand/10 rounded-lg">
                <FolderOpen className="w-8 h-8 text-brand" />
              </div>
              <div>
                <h1 className="text-3xl font-bold text-content">
                  {project?.name || "Unknown Project"}
                </h1>
                <div className="flex items-center gap-4 mt-1">
                  <span className="text-sm text-content-subtle font-mono bg-background-subtle px-2 py-1 rounded">
                    {project?.slug || "unknown-slug"}
                  </span>
                  {project?.gitRepositoryUrl && (
                    <a
                      href={project.gitRepositoryUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 text-sm text-content-subtle hover:text-content transition-colors"
                    >
                      <ExternalLink className="w-4 h-4" />
                      Repository
                      <ExternalLink className="w-3 h-3" />
                    </a>
                  )}
                </div>
              </div>
            </div>

            <div className="flex items-center gap-3">
              <Button variant="outline" size="md">
                <Settings className="w-4 h-4 mr-2" />
                Settings
              </Button>
              <Button variant="primary" size="md">
                <Plus className="w-4 h-4 mr-2" />
                Deploy
              </Button>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="border-b border-border mb-6">
          <nav className="flex space-x-8">
            {[
              { key: "overview", label: "Overview", icon: Activity },
              { key: "deployments", label: "Deployments", icon: Tag },
              { key: "settings", label: "Settings", icon: Settings },
            ].map(({ key, label, icon: Icon }) => (
              <button
                key={key}
                type="button"
                onClick={() => setActiveTab(key as typeof activeTab)}
                className={`flex items-center gap-2 py-2 px-1 border-b-2 font-medium text-sm transition-colors ${
                  activeTab === key
                    ? "border-brand text-brand"
                    : "border-transparent text-content-subtle hover:text-content hover:border-border"
                }`}
              >
                <Icon className="w-4 h-4" />
                {label}
              </button>
            ))}
          </nav>
        </div>

        {/* Tab Content */}
        {activeTab === "overview" && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Stats Cards */}
            <div className="lg:col-span-3 grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
              <div className="bg-white p-6 rounded-lg border border-border">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-content-subtle">Production</p>
                    <p className="text-2xl font-bold text-content">
                      {deployments.filter((d) => d.environment === "production").length}
                    </p>
                  </div>
                  <Globe className="w-8 h-8 text-brand" />
                </div>
              </div>

              <div className="bg-white p-6 rounded-lg border border-border">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-content-subtle">Total Deployments</p>
                    <p className="text-2xl font-bold text-content">{deployments.length}</p>
                  </div>
                  <Tag className="w-8 h-8 text-success" />
                </div>
              </div>

              <div className="bg-white p-6 rounded-lg border border-border">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-content-subtle">Active Deployments</p>
                    <p className="text-2xl font-bold text-content">
                      {deployments.filter((d) => d.status === "active").length}
                    </p>
                  </div>
                  <Globe className="w-8 h-8 text-brand" />
                </div>
              </div>

              <div className="bg-white p-6 rounded-lg border border-border">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-content-subtle">Last Updated</p>
                    <p className="text-sm font-bold text-content">
                      {project?.updatedAt || project?.createdAt
                        ? new Date(project.updatedAt || project.createdAt).toLocaleDateString()
                        : "Unknown"}
                    </p>
                  </div>
                  <Clock className="w-8 h-8 text-warn" />
                </div>
              </div>
            </div>

            {/* Recent Activity */}
            <div className="lg:col-span-2 bg-white rounded-lg border border-border p-6">
              <h3 className="text-lg font-semibold text-content mb-4">Recent Deployments</h3>
              {deployments.length > 0 ? (
                <div className="space-y-4">
                  {deployments.slice(0, 5).map((deployment) => (
                    <a
                      key={deployment.id}
                      href={`/projects/${projectId}/deployments/${deployment.id}`}
                      className="block"
                    >
                      <div className="flex items-center justify-between p-3 bg-background-subtle rounded-lg hover:bg-border transition-colors">
                        <div className="flex items-center gap-3">
                          <Tag className="w-4 h-4 text-content-subtle" />
                          <div>
                            <p className="font-medium text-content">{deployment.environment}</p>
                            <p className="text-sm text-content-subtle">
                              {deployment.gitCommitSha?.substring(0, 8)} •{" "}
                              {new Date(deployment.createdAt).toLocaleDateString()}
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <span
                            className={`px-2 py-1 text-xs font-medium rounded-full border ${getStatusColor(deployment.status)}`}
                          >
                            {deployment.status}
                          </span>
                          <ChevronRight className="w-4 h-4 text-content-subtle" />
                        </div>
                      </div>
                    </a>
                  ))}
                </div>
              ) : (
                <p className="text-content-subtle">No deployments found.</p>
              )}
            </div>

            {/* Quick Actions */}
            <div className="bg-white rounded-lg border border-border p-6">
              <h3 className="text-lg font-semibold text-content mb-4">Quick Actions</h3>
              <div className="space-y-3">
                <a href={`/projects/${projectId}/deploy`} className="block">
                  <Button variant="outline" size="md" className="w-full justify-start">
                    <Play className="w-4 h-4 mr-2" />
                    Deploy Latest
                  </Button>
                </a>
                <Button variant="outline" size="md" className="w-full justify-start">
                  <RotateCcw className="w-4 h-4 mr-2" />
                  Rollback
                </Button>
                <a href={`/projects/${projectId}/logs`} className="block">
                  <Button variant="outline" size="md" className="w-full justify-start">
                    <Eye className="w-4 h-4 mr-2" />
                    View Logs
                  </Button>
                </a>
                <a href={`/projects/${projectId}/diff`} className="block">
                  <Button variant="outline" size="md" className="w-full justify-start">
                    <GitCommit className="w-4 h-4 mr-2" />
                    Compare Deployments
                  </Button>
                </a>
                <a href={`/projects/${projectId}/settings`} className="block">
                  <Button variant="outline" size="md" className="w-full justify-start">
                    <Settings className="w-4 h-4 mr-2" />
                    Configure
                  </Button>
                </a>
              </div>
            </div>
          </div>
        )}

        {activeTab === "deployments" && (
          <div>
            {/* Search */}
            <div className="mb-6 flex flex-col sm:flex-row gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-content-subtle w-4 h-4" />
                <input
                  type="text"
                  placeholder={`Search ${activeTab}...`}
                  value={searchTerm}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setSearchTerm(e.target.value)
                  }
                  className="w-full pl-10 pr-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-brand focus:border-transparent bg-white text-content"
                />
              </div>
            </div>

            {/* Content */}
            <div>
              <div className="bg-white rounded-lg border border-border overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-border">
                    <thead className="bg-background-subtle">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                          Deployment
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                          Branch
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                          Environment
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                          Status
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                          Created
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                          Actions
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-border">
                      {filteredDeployments.map((deployment) => (
                        <tr key={deployment.id} className="hover:bg-background-subtle">
                          <td className="px-6 py-4 whitespace-nowrap">
                            <div className="flex items-center">
                              <GitCommit className="w-4 h-4 text-content-subtle mr-2" />
                              <span className="text-sm font-mono text-content">
                                {deployment.gitCommitSha}
                              </span>
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span className="text-sm text-content">{deployment.gitBranch}</span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span className="text-sm text-content capitalize">
                              {deployment.environment}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span
                              className={`px-2 py-1 text-xs font-medium rounded-full border ${getStatusColor(deployment.status)}`}
                            >
                              {deployment.status}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-content-subtle">
                            {new Date(deployment.createdAt).toLocaleDateString()}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                            <div className="flex items-center gap-2">
                              <a href={`/projects/${projectId}/deployments/${deployment.id}`}>
                                <Button variant="ghost" size="sm">
                                  <Eye className="w-3 h-3" />
                                </Button>
                              </a>
                              <Button variant="ghost" size="sm">
                                <RotateCcw className="w-3 h-3" />
                              </Button>
                              <Button variant="ghost" size="sm">
                                <MoreVertical className="w-3 h-3" />
                              </Button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === "settings" && (
          <div className="bg-white rounded-lg border border-border p-6">
            <h3 className="text-lg font-semibold text-content mb-4">Project Settings</h3>
            <p className="text-content-subtle">Settings configuration coming soon...</p>
          </div>
        )}
      </div>
    </div>
  );
}
