"use client";

import { trpc } from "@/lib/trpc/client";
import { Button } from "@unkey/ui";
import {
  Activity,
  ExternalLink,
  Eye,
  Filter,
  FolderOpen,
  FolderPlus,
  GitBranch,
  Github,
  Plus,
  Search,
  Tag,
} from "lucide-react";
import { useState } from "react";

// Type definitions
interface Project {
  id: string;
  name: string;
  slug: string;
  gitRepositoryUrl: string | null;
  createdAt: number;
  updatedAt: number | null;
}

export default function ProjectsPage(): JSX.Element {
  const [name, setName] = useState<string>("");
  const [slug, setSlug] = useState<string>("");
  const [gitUrl, setGitUrl] = useState<string>("");
  const [searchTerm, setSearchTerm] = useState<string>("");
  const [showCreateForm, setShowCreateForm] = useState<boolean>(false);

  // Use actual tRPC hooks
  const { data, isLoading, refetch } = trpc.project.list.useQuery();
  const createProject = trpc.project.create.useMutation({
    onSuccess: () => {
      refetch();
      setName("");
      setSlug("");
      setGitUrl("");
      setShowCreateForm(false);
    },
  });

  const handleSubmit = async (
    e: React.FormEvent<HTMLFormElement> | React.MouseEvent<HTMLButtonElement>,
  ): Promise<void> => {
    e.preventDefault();
    if (!name || !slug) {
      return;
    }

    createProject.mutate({
      name,
      slug,
      gitRepositoryUrl: gitUrl || undefined,
    });
  };

  // Get projects from tRPC data
  const projects: Project[] = data?.projects || [];
  const isCreating: boolean = createProject.isLoading;

  const filteredProjects: Project[] = projects.filter(
    (project: Project) =>
      project.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      project.slug.toLowerCase().includes(searchTerm.toLowerCase()),
  );

  const generateSlug = (name: string): string => {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "");
  };

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-content">Projects</h1>
              <p className="text-content-subtle mt-1">
                Manage your deployment projects and configurations
              </p>
            </div>
            <Button onClick={() => setShowCreateForm(true)} variant="primary" size="md">
              <Plus className="w-4 h-4 mr-2" />
              New Project
            </Button>
          </div>
        </div>

        {/* Search and Filters */}
        <div className="mb-6 flex flex-col sm:flex-row gap-4">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-content-subtle w-4 h-4" />
            <input
              type="text"
              placeholder="Search projects..."
              value={searchTerm}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-brand focus:border-transparent bg-white text-content"
            />
          </div>
          <Button variant="outline" size="md">
            <Filter className="w-4 h-4 mr-2" />
            Filter
          </Button>
        </div>

        {/* Error Display */}
        {createProject.error && (
          <div className="mb-6 bg-alert/10 border border-alert/20 rounded-lg p-4">
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-alert" viewBox="0 0 20 20" fill="currentColor">
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                    clipRule="evenodd"
                  />
                </svg>
              </div>
              <div className="ml-3">
                <h3 className="text-sm font-medium text-alert">Error creating project</h3>
                <div className="mt-2 text-sm text-alert">{createProject.error.message}</div>
              </div>
            </div>
          </div>
        )}

        {/* Create Project Form */}
        {showCreateForm && (
          <div className="mb-8 bg-white rounded-xl shadow-sm border border-border p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-content">Create New Project</h2>
              <Button onClick={() => setShowCreateForm(false)} variant="ghost" size="sm">
                ✕
              </Button>
            </div>

            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label
                    htmlFor="project-name"
                    className="block text-sm font-medium text-content-subtle mb-1"
                  >
                    Project Name *
                  </label>
                  <input
                    id="project-name"
                    type="text"
                    value={name}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      setName(e.target.value);
                      setSlug(generateSlug(e.target.value));
                    }}
                    placeholder="Enter project name"
                    className="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-brand focus:border-transparent bg-white text-content"
                    required
                  />
                </div>

                <div>
                  <label
                    htmlFor="project-slug"
                    className="block text-sm font-medium text-content-subtle mb-1"
                  >
                    Project Slug *
                  </label>
                  <input
                    id="project-slug"
                    type="text"
                    value={slug}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSlug(e.target.value)}
                    placeholder="project-slug"
                    className="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-brand focus:border-transparent bg-white text-content"
                    required
                  />
                </div>
              </div>

              <div>
                <label
                  htmlFor="git-repo-url"
                  className="block text-sm font-medium text-content-subtle mb-1"
                >
                  Git Repository URL (optional)
                </label>
                <div className="relative">
                  <Github className="absolute left-3 top-1/2 transform -translate-y-1/2 text-content-subtle w-4 h-4" />
                  <input
                    id="git-repo-url"
                    type="url"
                    value={gitUrl}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setGitUrl(e.target.value)}
                    placeholder="https://github.com/username/repository"
                    className="w-full pl-10 pr-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-brand focus:border-transparent bg-white text-content"
                  />
                </div>
              </div>

              <div className="flex items-center gap-3 pt-2">
                <Button
                  onClick={handleSubmit}
                  disabled={!name || !slug}
                  loading={isCreating}
                  variant="primary"
                  size="md"
                >
                  <FolderPlus className="w-4 h-4 mr-2" />
                  Create Project
                </Button>
                <Button onClick={() => setShowCreateForm(false)} variant="ghost" size="md">
                  Cancel
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Projects Grid */}
        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5 w-full p-5">
            {Array.from({ length: 8 }, (_, i) => ({ id: `skeleton-${i}` })).map((item) => (
              <div
                key={item.id}
                className="bg-white rounded-xl shadow-sm border border-border p-6 animate-pulse"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex-1">
                    <div className="h-5 bg-background-subtle rounded mb-2 w-3/4" />
                    <div className="h-4 bg-background-subtle rounded w-1/2" />
                  </div>
                  <div className="w-5 h-5 bg-background-subtle rounded" />
                </div>
                <div className="h-16 bg-background-subtle rounded mb-4" />
                <div className="space-y-2 mb-4">
                  <div className="h-3 bg-background-subtle rounded w-full" />
                  <div className="h-3 bg-background-subtle rounded w-2/3" />
                </div>
                <div className="h-8 bg-background-subtle rounded" />
              </div>
            ))}
          </div>
        ) : filteredProjects.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5 w-full p-5">
            {filteredProjects.map((project) => (
              <div
                key={project.id}
                className="bg-white rounded-xl shadow-sm border border-border hover:shadow-md transition-all duration-200 overflow-hidden group"
              >
                <a href={`/projects/${project.id}`} className="block">
                  <div className="p-6">
                    {/* Header */}
                    <div className="flex items-start justify-between mb-4">
                      <div className="flex-1 min-w-0">
                        <h3 className="text-lg font-semibold text-content mb-1 group-hover:text-brand transition-colors truncate">
                          {project.name}
                        </h3>
                        <p className="text-xs text-content-subtle font-mono bg-background-subtle px-2 py-1 rounded inline-block">
                          {project.slug}
                        </p>
                      </div>
                      <div className="flex items-center space-x-2 flex-shrink-0 ml-3">
                        <FolderOpen className="w-5 h-5 text-brand" />
                        {project.gitRepositoryUrl && (
                          <a
                            href={project.gitRepositoryUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-content-subtle hover:text-content transition-colors"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <Github className="w-4 h-4" />
                          </a>
                        )}
                      </div>
                    </div>

                    {/* Chart/Visual Area */}
                    <div className="mb-4 h-16 bg-gradient-to-r from-brand/5 to-brand/10 rounded-lg flex items-center justify-center">
                      <div className="flex items-center gap-2 text-content-subtle">
                        <Activity className="w-4 h-4" />
                        <span className="text-sm">Activity metrics coming soon</span>
                      </div>
                    </div>

                    {/* Stats */}
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-4">
                        <div className="flex items-center gap-1">
                          <GitBranch className="w-3 h-3 text-content-subtle" />
                          <span className="text-xs text-content-subtle">0 Branches</span>
                        </div>
                        <div className="flex items-center gap-1">
                          <Tag className="w-3 h-3 text-content-subtle" />
                          <span className="text-xs text-content-subtle">0 Versions</span>
                        </div>
                      </div>
                      <div className="text-xs text-content-subtle">
                        {new Date(project.createdAt).toLocaleDateString()}
                      </div>
                    </div>

                    {/* Repository info */}
                    {project.gitRepositoryUrl && (
                      <div className="flex items-center text-xs text-content-subtle mb-4 truncate">
                        <Github className="w-3 h-3 mr-1 flex-shrink-0" />
                        <span className="truncate">
                          {project.gitRepositoryUrl.replace("https://github.com/", "")}
                        </span>
                      </div>
                    )}
                  </div>
                </a>

                {/* Footer Actions */}
                <div className="px-6 pb-6">
                  <div className="flex items-center justify-between pt-4 border-t border-border">
                    <a
                      href={`/projects/${project.id}`}
                      className="inline-flex items-center text-brand hover:text-brand/80 font-medium transition-colors duration-200 text-sm"
                    >
                      <Eye className="w-3 h-3 mr-1" />
                      View Project
                      <ExternalLink className="w-3 h-3 ml-1" />
                    </a>
                    <div className="flex items-center gap-1">
                      <div className="w-2 h-2 bg-success rounded-full" />
                      <span className="text-xs text-content-subtle">Active</span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center py-16 px-4">
            <div className="text-center max-w-md">
              <div className="w-16 h-16 mx-auto mb-6 bg-background-subtle rounded-full flex items-center justify-center">
                <FolderPlus className="w-8 h-8 text-content-subtle" />
              </div>
              <h3 className="text-xl font-semibold text-content mb-2">
                {searchTerm ? "No projects found" : "No projects yet"}
              </h3>
              <p className="text-content-subtle mb-8 leading-relaxed">
                {searchTerm
                  ? `No projects match "${searchTerm}". Try adjusting your search criteria.`
                  : "Create your first project to start deploying APIs globally with predictable performance."}
              </p>
              {!searchTerm && (
                <div className="flex flex-col sm:flex-row gap-3 justify-center">
                  <Button onClick={() => setShowCreateForm(true)} variant="primary" size="md">
                    <Plus className="w-4 h-4 mr-2" />
                    Create Your First Project
                  </Button>
                  <a
                    href="https://docs.unkey.com"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-2 px-4 py-2 border border-border rounded-lg text-content hover:bg-background-subtle transition-colors duration-200 font-medium text-sm"
                  >
                    <ExternalLink className="w-4 h-4" />
                    Documentation
                  </a>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
