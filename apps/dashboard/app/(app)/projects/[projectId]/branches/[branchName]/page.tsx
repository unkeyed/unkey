"use client";

import { Button } from "@unkey/ui";
import {
  Activity,
  AlertCircle,
  ArrowLeft,
  CheckCircle,
  Clock,
  Code,
  ExternalLink,
  Eye,
  GitBranch,
  GitCommit,
  Globe,
  Loader,
  MoreVertical,
  Play,
  RotateCcw,
  Settings,
  Shield,
  Tag,
  Terminal,
  XCircle,
  Zap,
} from "lucide-react";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";

import { trpc } from "@/lib/trpc/client";

// Type definitions
interface Project {
  id: string;
  name: string;
  slug: string;
}

interface Environment {
  id: string;
  name: string;
  description?: string;
}

interface Branch {
  id: string;
  name: string;
  projectId: string;
  environmentId: string;
  isProduction: boolean;
  createdAt: number;
  updatedAt: number | null;
  environment: Environment;
  lastCommitSha?: string;
  lastCommitMessage?: string;
  lastCommitAuthor?: string;
  lastCommitDate?: number;
}

interface Version {
  id: string;
  gitCommitSha: string | null;
  gitBranch: string | null;
  gitCommitMessage?: string;
  status: "pending" | "building" | "deploying" | "active" | "failed" | "archived";
  createdAt: number;
  updatedAt: number | null;
  buildDuration?: number;
  deploymentUrl?: string;
}

interface EnvironmentVariable {
  key: string;
  value: string;
  isSecret: boolean;
  source: "project" | "environment" | "branch";
}


// Mock data with static timestamps to avoid hydration issues
const mockProject: Project = {
  id: "proj_123",
  name: "Meg's Demo",
  slug: "megs-demo",
};

const mockBranch: Branch = {
  id: "branch_123",
  name: "feature/auth-improvements",
  projectId: "proj_123",
  environmentId: "env_preview",
  isProduction: false,
  createdAt: 1721044800000, // Static timestamp
  updatedAt: 1721048400000, // Static timestamp
  environment: {
    id: "env_preview",
    name: "preview",
    description: "Preview environment for feature testing",
  },
  lastCommitSha: "a1b2c3d",
  lastCommitMessage: "Add OAuth2 integration with refresh token support",
  lastCommitAuthor: "john.doe",
  lastCommitDate: 1721048400000, // Static timestamp
};

const mockVersions: Version[] = [
  {
    id: "ver_1",
    gitCommitSha: "a1b2c3d",
    gitBranch: "feature/auth-improvements",
    gitCommitMessage: "Add OAuth2 integration with refresh token support",
    status: "active",
    createdAt: 1721048400000, // Static timestamp
    updatedAt: 1721048400000, // Static timestamp
    buildDuration: 180,
    deploymentUrl: "https://a1b2c3d-megs-demo.unkey.app",
  },
  {
    id: "ver_2",
    gitCommitSha: "x9y8z7w",
    gitBranch: "feature/auth-improvements",
    gitCommitMessage: "Fix token validation edge cases",
    status: "archived",
    createdAt: 1720962000000, // Static timestamp
    updatedAt: 1720962000000, // Static timestamp
    buildDuration: 165,
  },
  {
    id: "ver_3",
    gitCommitSha: "m5n6o7p",
    gitBranch: "feature/auth-improvements",
    gitCommitMessage: "Initial OAuth implementation",
    status: "failed",
    createdAt: 1720875600000, // Static timestamp
    updatedAt: 1720875600000, // Static timestamp
    buildDuration: 45,
  },
];

const mockEnvVars: EnvironmentVariable[] = [
  {
    key: "API_URL",
    value: "https://api.preview.unkey.app",
    isSecret: false,
    source: "environment",
  },
  { key: "DATABASE_URL", value: "***", isSecret: true, source: "environment" },
  { key: "FEATURE_FLAG_AUTH", value: "true", isSecret: false, source: "branch" },
  { key: "DEBUG_MODE", value: "true", isSecret: false, source: "branch" },
];

export default function BranchDetailPage(): JSX.Element {
  const params = useParams();
  const router = useRouter();
  const projectId = params?.projectId as string;
  const branchName = params?.branchName as string;

  const [activeTab, setActiveTab] = useState<"overview" | "versions" | "config" | "settings">(
    "overview",
  );
  const [showEnvVars, setShowEnvVars] = useState(false);

  // tRPC calls
  const { data: branchData, isLoading } = trpc.branch.getByName.useQuery({ projectId, branchName });
  // const { data: versionsData } = trpc.version.listByBranch.useQuery({ projectId, branchName });

  // Use real data or fallback to mock data while loading
  const project = branchData?.project || mockProject;
  const branch = branchData || mockBranch;
  const versions = branchData?.versions || mockVersions;
  const envVars = mockEnvVars; // Still using mock data for env vars as it's not in the query

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "active":
        return <CheckCircle className="w-4 h-4 text-success" />;
      case "failed":
        return <XCircle className="w-4 h-4 text-alert" />;
      case "building":
      case "deploying":
      case "pending":
        return <Loader className="w-4 h-4 text-warn animate-spin" />;
      case "archived":
        return <Clock className="w-4 h-4 text-content-subtle" />;
      default:
        return <AlertCircle className="w-4 h-4 text-content-subtle" />;
    }
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case "active":
        return "text-success bg-success/10 border-success/20";
      case "building":
      case "deploying":
      case "pending":
        return "text-warn bg-warn/10 border-warn/20";
      case "failed":
        return "text-alert bg-alert/10 border-alert/20";
      case "archived":
        return "text-content-subtle bg-background-subtle border-border";
      default:
        return "text-content-subtle bg-background-subtle border-border";
    }
  };

  const activeVersion = versions.find((v) => v.status === "active");

  // Navigation handlers for compare functionality
  const handleCompareBranchVersions = () => {
    // Get the two most recent versions if available
    if (versions.length >= 2) {
      const [latest, previous] = versions;
      router.push(`/projects/${projectId}/diff/${previous.id}/${latest.id}`);
    } else {
      alert("At least two versions are needed to compare. Please create more versions first.");
    }
  };

  const handleCompareWithOtherBranches = () => {
    // Navigate back to project page to select branches for comparison
    router.push(`/projects/${projectId}/diff`);
  };

  const handleCompareFromVersion = (versionId: string) => {
    // Compare with the previous version if available
    const currentIndex = versions.findIndex(v => v.id === versionId);
    const previousVersion = versions[currentIndex + 1];
    
    if (previousVersion) {
      router.push(`/projects/${projectId}/diff/${previousVersion.id}/${versionId}`);
    } else {
      alert("No previous version found to compare with.");
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="flex items-center gap-3">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand"></div>
          <span className="text-content-subtle">Loading branch...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-4 mb-4">
            <a
              href={`/projects/${projectId}`}
              className="inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-content-subtle hover:text-content transition-colors rounded-md hover:bg-background-subtle"
            >
              <ArrowLeft className="w-4 h-4" />
              Back to {project.name}
            </a>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-brand/10 rounded-lg">
                <GitBranch className="w-8 h-8 text-brand" />
              </div>
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <h1 className="text-3xl font-bold text-content">{branch.name}</h1>
                  {branch.isProduction && (
                    <span className="px-3 py-1 text-sm font-medium bg-success/10 text-success border border-success/20 rounded-full">
                      Production
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-4 text-sm text-content-subtle">
                  <span className="bg-background-subtle px-2 py-1 rounded font-mono">
                    {branch.environment.name}
                  </span>
                  {branch.lastCommitSha && (
                    <div className="flex items-center gap-1">
                      <GitCommit className="w-3 h-3" />
                      <span className="font-mono">{branch.lastCommitSha}</span>
                    </div>
                  )}
                  {activeVersion?.deploymentUrl && (
                    <a
                      href={activeVersion.deploymentUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 hover:text-content transition-colors"
                    >
                      <Globe className="w-3 h-3" />
                      Live deployment
                      <ExternalLink className="w-3 h-3" />
                    </a>
                  )}
                </div>
              </div>
            </div>

            <div className="flex items-center gap-3">
              <Button variant="outline" size="md">
                <Settings className="w-4 h-4 mr-2" />
                Configure
              </Button>
              <Button variant="primary" size="md">
                <Play className="w-4 h-4 mr-2" />
                Deploy
              </Button>
            </div>
          </div>
        </div>

        {/* Latest Commit Info */}
        {branch.lastCommitMessage && (
          <div className="mb-6 bg-white rounded-lg border border-border p-4">
            <div className="flex items-start gap-3">
              <GitCommit className="w-5 h-5 text-content-subtle mt-0.5" />
              <div className="flex-1 min-w-0">
                <p className="text-content font-medium mb-1">{branch.lastCommitMessage}</p>
                <div className="flex items-center gap-4 text-sm text-content-subtle">
                  <span>{branch.lastCommitAuthor}</span>
                  <span>
                    {branch.lastCommitDate && new Date(branch.lastCommitDate).toLocaleString()}
                  </span>
                  <span className="font-mono">{branch.lastCommitSha}</span>
                </div>
              </div>
              {activeVersion && (
                <div className="flex items-center gap-2">
                  {getStatusIcon(activeVersion.status)}
                  <span
                    className={`px-2 py-1 text-xs font-medium rounded-full border ${getStatusColor(activeVersion.status)}`}
                  >
                    {activeVersion.status}
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Tabs */}
        <div className="border-b border-border mb-6">
          <nav className="flex space-x-8">
            {[
              { key: "overview", label: "Overview", icon: Activity },
              { key: "versions", label: "Versions", icon: Tag },
              { key: "config", label: "Configuration", icon: Settings },
              { key: "settings", label: "Branch Settings", icon: Shield },
            ].map(({ key, label, icon: Icon }) => (
              <button
                key={key}
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
            {/* Deployment Status */}
            <div className="lg:col-span-2 space-y-6">
              {/* Active Deployment */}
              {activeVersion && (
                <div className="bg-white rounded-lg border border-border p-6">
                  <h3 className="text-lg font-semibold text-content mb-4">Active Deployment</h3>
                  <div className="flex items-center justify-between p-4 bg-success/5 border border-success/20 rounded-lg">
                    <div className="flex items-center gap-3">
                      <CheckCircle className="w-5 h-5 text-success" />
                      <div>
                        <p className="font-medium text-content">
                          Version {activeVersion.gitCommitSha}
                        </p>
                        <p className="text-sm text-content-subtle">
                          Deployed {new Date(activeVersion.createdAt).toLocaleString()}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button variant="outline" size="sm">
                        <Eye className="w-3 h-3 mr-1" />
                        View
                      </Button>
                      <Button variant="outline" size="sm">
                        <RotateCcw className="w-3 h-3 mr-1" />
                        Rollback
                      </Button>
                    </div>
                  </div>
                </div>
              )}

              {/* Recent Activity */}
              <div className="bg-white rounded-lg border border-border p-6">
                <h3 className="text-lg font-semibold text-content mb-4">Recent Activity</h3>
                <div className="space-y-3">
                  {versions.slice(0, 3).map((version) => (
                    <div
                      key={version.id}
                      className="flex items-center justify-between p-3 bg-background-subtle rounded-lg"
                    >
                      <div className="flex items-center gap-3">
                        {getStatusIcon(version.status)}
                        <div>
                          <p className="font-medium text-content">
                            {version.gitCommitMessage || `Version ${version.gitCommitSha}`}
                          </p>
                          <p className="text-sm text-content-subtle">
                            {new Date(version.createdAt).toLocaleString()}
                            {version.buildDuration && ` • Build: ${version.buildDuration}s`}
                          </p>
                        </div>
                      </div>
                      <span
                        className={`px-2 py-1 text-xs font-medium rounded-full border ${getStatusColor(version.status)}`}
                      >
                        {version.status}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Sidebar */}
            <div className="space-y-6">
              {/* Quick Stats */}
              <div className="bg-white rounded-lg border border-border p-6">
                <h3 className="text-lg font-semibold text-content mb-4">Quick Stats</h3>
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <span className="text-content-subtle">Total Versions</span>
                    <span className="font-semibold text-content">{versions.length}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-content-subtle">Success Rate</span>
                    <span className="font-semibold text-success">
                      {Math.round(
                        (versions.filter((v) => v.status === "active" || v.status === "archived")
                          .length /
                          versions.length) *
                          100,
                      )}
                      %
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-content-subtle">Avg Build Time</span>
                    <span className="font-semibold text-content">
                      {Math.round(
                        versions
                          .filter((v) => v.buildDuration)
                          .reduce((acc, v) => acc + (v.buildDuration || 0), 0) /
                          versions.filter((v) => v.buildDuration).length,
                      )}
                      s
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-content-subtle">Environment</span>
                    <span className="font-semibold text-content">{branch.environment.name}</span>
                  </div>
                </div>
              </div>

              {/* Quick Actions */}
              <div className="bg-white rounded-lg border border-border p-6">
                <h3 className="text-lg font-semibold text-content mb-4">Quick Actions</h3>
                <div className="space-y-3">
                  <Button
                    variant="outline"
                    size="md"
                    className="w-full justify-start"
                    onClick={handleCompareWithOtherBranches}
                  >
                    <GitBranch className="w-4 h-4 mr-2" />
                    Compare API Versions
                  </Button>
                  <Button variant="outline" size="md" className="w-full justify-start">
                    <Terminal className="w-4 h-4 mr-2" />
                    View Logs
                  </Button>
                  <Button variant="outline" size="md" className="w-full justify-start">
                    <Code className="w-4 h-4 mr-2" />
                    Open in IDE
                  </Button>
                  <Button variant="outline" size="md" className="w-full justify-start">
                    <GitBranch className="w-4 h-4 mr-2" />
                    View on GitHub
                  </Button>
                  <Button variant="outline" size="md" className="w-full justify-start">
                    <Zap className="w-4 h-4 mr-2" />
                    Trigger Build
                  </Button>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === "versions" && (
          <div className="bg-white rounded-lg border border-border overflow-hidden">
            {/* OPTION 1: Enhanced header with compare buttons */}
            <div className="p-6 border-b border-border">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-semibold text-content">Version History</h3>
                  <p className="text-content-subtle mt-1">All deployments for this branch</p>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="md" onClick={handleCompareBranchVersions}>
                    <GitBranch className="w-4 h-4 mr-2" />
                    Compare Versions
                  </Button>
                  <Button variant="outline" size="md" onClick={handleCompareWithOtherBranches}>
                    <GitBranch className="w-4 h-4 mr-2" />
                    Compare with Other Branches
                  </Button>
                </div>
              </div>
            </div>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-border">
                <thead className="bg-background-subtle">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                      Version
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                      Commit
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-content-subtle uppercase tracking-wider">
                      Build Time
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
                  {versions.map((version) => (
                    <tr key={version.id} className="hover:bg-background-subtle">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <span className="text-sm font-mono text-content">{version.id}</span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div>
                          <div className="text-sm font-mono text-content">
                            {version.gitCommitSha}
                          </div>
                          {version.gitCommitMessage && (
                            <div className="text-sm text-content-subtle truncate max-w-xs">
                              {version.gitCommitMessage}
                            </div>
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-2">
                          {getStatusIcon(version.status)}
                          <span
                            className={`px-2 py-1 text-xs font-medium rounded-full border ${getStatusColor(version.status)}`}
                          >
                            {version.status}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-content">
                        {version.buildDuration ? `${version.buildDuration}s` : "-"}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-content-subtle">
                        {new Date(version.createdAt).toLocaleString()}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                        <div className="flex items-center gap-2">
                          <Button variant="ghost" size="sm" title="View Details">
                            <Eye className="w-3 h-3" />
                          </Button>
                          {version.status !== "active" && (
                            <Button variant="ghost" size="sm" title="Deploy">
                              <Play className="w-3 h-3" />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="sm"
                            title="Compare this version"
                            onClick={() => handleCompareFromVersion(version.id)}
                          >
                            <GitBranch className="w-3 h-3" />
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
        )}

        {activeTab === "config" && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Environment Variables */}
            <div className="bg-white rounded-lg border border-border p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-content">Environment Variables</h3>
                <Button variant="outline" size="sm" onClick={() => setShowEnvVars(!showEnvVars)}>
                  {showEnvVars ? "Hide Values" : "Show Values"}
                </Button>
              </div>
              <div className="space-y-3">
                {envVars.map((envVar) => (
                  <div
                    key={envVar.key}
                    className="flex items-center justify-between p-3 bg-background-subtle rounded-lg"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-sm text-content">{envVar.key}</span>
                        <span
                          className={`px-1.5 py-0.5 text-xs rounded ${
                            envVar.source === "branch"
                              ? "bg-brand/10 text-brand"
                              : envVar.source === "environment"
                                ? "bg-warn/10 text-warn"
                                : "bg-content-subtle/10 text-content-subtle"
                          }`}
                        >
                          {envVar.source}
                        </span>
                      </div>
                      <div className="text-sm text-content-subtle font-mono mt-1">
                        {envVar.isSecret && !showEnvVars ? "***" : envVar.value}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Environment Configuration */}
            <div className="bg-white rounded-lg border border-border p-6">
              <h3 className="text-lg font-semibold text-content mb-4">Environment Configuration</h3>
              <div className="space-y-4">
                <div className="flex items-center justify-between p-3 bg-background-subtle rounded-lg">
                  <span className="text-content-subtle">Current Environment</span>
                  <span className="font-semibold text-content">{branch.environment.name}</span>
                </div>
                <div className="flex items-center justify-between p-3 bg-background-subtle rounded-lg">
                  <span className="text-content-subtle">Production Branch</span>
                  <span
                    className={`px-2 py-1 text-xs font-medium rounded-full ${
                      branch.isProduction
                        ? "bg-success/10 text-success"
                        : "bg-content-subtle/10 text-content-subtle"
                    }`}
                  >
                    {branch.isProduction ? "Yes" : "No"}
                  </span>
                </div>
                <div className="flex items-center justify-between p-3 bg-background-subtle rounded-lg">
                  <span className="text-content-subtle">Auto-deploy</span>
                  <span className="px-2 py-1 text-xs font-medium rounded-full bg-success/10 text-success">
                    Enabled
                  </span>
                </div>
                <div className="flex items-center justify-between p-3 bg-background-subtle rounded-lg">
                  <span className="text-content-subtle">Branch Protection</span>
                  <span className="px-2 py-1 text-xs font-medium rounded-full bg-content-subtle/10 text-content-subtle">
                    None
                  </span>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === "settings" && (
          <div className="max-w-2xl">
            <div className="bg-white rounded-lg border border-border p-6">
              <h3 className="text-lg font-semibold text-content mb-4">Branch Settings</h3>
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-medium text-content-subtle mb-2">
                    Environment Assignment
                  </label>
                  <select className="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-brand focus:border-transparent bg-white text-content">
                    <option value="preview">Preview</option>
                    <option value="staging">Staging</option>
                    <option value="production">Production</option>
                  </select>
                </div>

                <div>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={branch.isProduction}
                      className="rounded border-border text-brand focus:ring-brand"
                    />
                    <span className="text-sm font-medium text-content">Production Branch</span>
                  </label>
                  <p className="text-sm text-content-subtle mt-1">
                    Production branches have additional safeguards and monitoring
                  </p>
                </div>

                <div>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      defaultChecked
                      className="rounded border-border text-brand focus:ring-brand"
                    />
                    <span className="text-sm font-medium text-content">Auto-deploy on push</span>
                  </label>
                  <p className="text-sm text-content-subtle mt-1">
                    Automatically deploy when new commits are pushed to this branch
                  </p>
                </div>

                <div className="pt-4 border-t border-border">
                  <h4 className="font-medium text-content mb-3">Danger Zone</h4>
                  <Button variant="outline" color="danger" size="md">
                    Delete Branch Configuration
                  </Button>
                  <p className="text-sm text-content-subtle mt-2">
                    This will remove the branch configuration but keep the Git branch intact
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
