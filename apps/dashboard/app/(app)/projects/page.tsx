"use client";

import { trpc } from "@/lib/trpc/client";
import { useState } from "react";

export default function ProjectsPage() {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [gitUrl, setGitUrl] = useState("");

  const { data, isLoading, refetch } = trpc.project.list.useQuery();
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

  return (
    <div style={{ padding: "20px" }}>
      <h1>Projects</h1>

      <h2>Existing Projects</h2>
      {isLoading ? (
        <p>Loading...</p>
      ) : data?.projects && data.projects.length > 0 ? (
        <div>
          {data.projects.map((project) => (
            <div
              key={project.id}
              style={{
                marginBottom: "20px",
                padding: "15px",
                border: "1px solid #ddd",
                borderRadius: "4px",
              }}
            >
              <h3>{project.name}</h3>
              <p>Slug: {project.slug}</p>
              <p>ID: {project.id}</p>
              {project.gitRepositoryUrl && <p>Git: {project.gitRepositoryUrl}</p>}
              <p>Created: {new Date(project.createdAt).toLocaleString()}</p>
              <a
                href={`/projects/${project.id}/branches`}
                style={{ color: "#0070f3", textDecoration: "underline" }}
              >
                View Branches →
              </a>
            </div>
          ))}
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
          }}
        >
          {createProject.isLoading ? "Creating..." : "Create Project"}
        </button>
      </form>

      {createProject.error && <p style={{ color: "red" }}>Error: {createProject.error.message}</p>}
    </div>
  );
}
