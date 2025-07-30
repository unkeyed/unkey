"use client";
import { trpc } from "@/lib/trpc/client";
import { useMemo, useState } from "react";

function useProjectsQuery() {
  const [nameFilter, setNameFilter] = useState("");

  const queryParams = useMemo(
    () => ({
      ...(nameFilter && {
        name: [{ operator: "contains" as const, value: nameFilter }],
      }),
    }),
    [nameFilter],
  );

  const { data, hasNextPage, fetchNextPage, isFetchingNextPage, isLoading, refetch } =
    trpc.deploy.project.query.useInfiniteQuery(queryParams, {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      staleTime: 30000, // 30 seconds
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    });

  const projects = useMemo(() => {
    if (!data?.pages) {
      return [];
    }
    return data.pages.flatMap((page) => page.projects);
  }, [data]);

  const total = data?.pages[0]?.total ?? 0;

  return {
    projects,
    isLoading,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    total,
    refetch,
    // Filter setters
    setNameFilter,
    nameFilter,
  };
}

export default function ProjectsPage() {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [gitUrl, setGitUrl] = useState("");

  const {
    projects,
    isLoading,
    hasMore,
    loadMore,
    isLoadingMore,
    total,
    refetch,
    setNameFilter,
    nameFilter,
  } = useProjectsQuery();

  const createProject = trpc.project.create.useMutation({
    onSuccess: () => {
      refetch();
      setName("");
      setSlug("");
      setGitUrl("");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
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

  const handleLoadMore = () => {
    if (hasMore && !isLoadingMore) {
      loadMore();
    }
  };

  const clearFilters = () => {
    setNameFilter("");
  };

  return (
    <div style={{ padding: "20px" }}>
      <h1>Projects</h1>

      {/* Filters */}
      <div
        style={{
          marginBottom: "20px",
          padding: "15px",
          backgroundColor: "#f5f5f5",
          borderRadius: "4px",
        }}
      >
        <h3>Filters</h3>
        <div style={{ display: "flex", gap: "10px", marginBottom: "10px" }}>
          <input
            type="text"
            placeholder="Filter by name..."
            value={nameFilter}
            onChange={(e) => setNameFilter(e.target.value)}
            style={{ padding: "5px", flex: 1 }}
          />
        </div>
        <button
          type="button"
          onClick={clearFilters}
          style={{
            padding: "5px 10px",
            backgroundColor: "#666",
            color: "white",
            border: "none",
            borderRadius: "4px",
            fontSize: "12px",
          }}
        >
          Clear Filters
        </button>
      </div>

      <h2>Existing Projects ({total} total)</h2>

      {isLoading ? (
        <p>Loading...</p>
      ) : projects.length > 0 ? (
        <div>
          {projects.map((project) => (
            <div
              key={project.id}
              style={{
                marginBottom: "15px",
                padding: "15px",
                border: "1px solid #ddd",
                borderRadius: "4px",
                backgroundColor: "#fff",
              }}
            >
              <h3>{project.name}</h3>
              <p>
                <strong>Slug:</strong> {project.slug}
              </p>
              <p>
                <strong>ID:</strong> {project.id}
              </p>
              {project.gitRepositoryUrl && (
                <p>
                  <strong>Git:</strong> {project.gitRepositoryUrl}
                </p>
              )}
              <p>
                <strong>Branch:</strong> {project.branch || "main"}
              </p>
              <p>
                <strong>Created:</strong> {new Date(project.createdAt).toLocaleString()}
              </p>
              {project.updatedAt && (
                <p>
                  <strong>Updated:</strong> {new Date(project.updatedAt).toLocaleString()}
                </p>
              )}
              <a
                href={`/projects/${project.id}/branches`}
                style={{ color: "#0070f3", textDecoration: "underline" }}
              >
                View Branches →
              </a>
            </div>
          ))}

          {/* Load More Button */}
          {hasMore && (
            <div style={{ textAlign: "center", marginTop: "20px" }}>
              <button
                type="button"
                onClick={handleLoadMore}
                disabled={isLoadingMore}
                style={{
                  padding: "10px 20px",
                  backgroundColor: "#0070f3",
                  color: "white",
                  border: "none",
                  borderRadius: "4px",
                  cursor: isLoadingMore ? "not-allowed" : "pointer",
                }}
              >
                {isLoadingMore ? "Loading..." : "Load More Projects"}
              </button>
            </div>
          )}
        </div>
      ) : (
        <p>No projects found.</p>
      )}

      <h2>Create New Project</h2>
      <form onSubmit={handleSubmit} style={{ marginTop: "20px" }}>
        <div style={{ marginBottom: "10px" }}>
          <label>
            Name:
            <input
              type="text"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                // Auto-generate slug
                const slug = e.target.value
                  .toLowerCase()
                  .replace(/[^a-z0-9\s-]/g, "")
                  .replace(/\s+/g, "-")
                  .replace(/-+/g, "-")
                  .replace(/^-|-$/g, "");
                setSlug(slug);
              }}
              style={{ marginLeft: "10px", padding: "5px" }}
            />
          </label>
        </div>
        <div style={{ marginBottom: "10px" }}>
          <label>
            Slug:
            <input
              type="text"
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              style={{ marginLeft: "10px", padding: "5px" }}
            />
          </label>
        </div>
        <div style={{ marginBottom: "10px" }}>
          <label>
            Git URL (optional):
            <input
              type="text"
              value={gitUrl}
              onChange={(e) => setGitUrl(e.target.value)}
              style={{ marginLeft: "10px", padding: "5px" }}
            />
          </label>
        </div>
        <button
          type="submit"
          disabled={!name || !slug || createProject.isLoading}
          style={{
            padding: "10px 20px",
            backgroundColor: "#0070f3",
            color: "white",
            border: "none",
            borderRadius: "4px",
            cursor: !name || !slug || createProject.isLoading ? "not-allowed" : "pointer",
          }}
        >
          {createProject.isLoading ? "Creating..." : "Create Project"}
        </button>
      </form>

      {createProject.error && (
        <p style={{ color: "red", marginTop: "10px" }}>Error: {createProject.error.message}</p>
      )}
    </div>
  );
}
