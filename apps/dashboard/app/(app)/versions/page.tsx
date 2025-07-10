"use client";

import { trpc } from "@/lib/trpc/client";

export default function VersionsPage() {
  const { data, isLoading, error } = trpc.version.list.useQuery();

  if (isLoading) {
    return (
      <div style={{ padding: "20px" }}>
        <p>Loading...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: "20px" }}>
        <h1>Error</h1>
        <p style={{ color: "red" }}>Failed to load versions: {error.message}</p>
      </div>
    );
  }

  return (
    <div style={{ padding: "20px" }}>
      <h1>All Versions</h1>

      {data?.versions && data.versions.length > 0 ? (
        <div>
          {data.versions.map((version) => (
            <div
              key={version.id}
              style={{
                marginBottom: "20px",
                padding: "15px",
                border: "1px solid #ddd",
                borderRadius: "4px",
              }}
            >
              <h3>Version {version.id}</h3>
              <p>
                <strong>Status:</strong> {version.status}
              </p>
              {version.gitCommitSha && (
                <p>
                  <strong>Git Commit:</strong> {version.gitCommitSha}
                </p>
              )}
              {version.gitBranch && (
                <p>
                  <strong>Git Branch:</strong> {version.gitBranch}
                </p>
              )}
              {version.rootfsImageId && (
                <p>
                  <strong>Rootfs Image:</strong> {version.rootfsImageId}
                </p>
              )}
              {version.buildId && (
                <p>
                  <strong>Build ID:</strong> {version.buildId}
                </p>
              )}
              {version.project && (
                <p>
                  <strong>Project:</strong> {version.project.name} ({version.project.slug})
                </p>
              )}
              {version.branch && (
                <p>
                  <strong>Branch:</strong> {version.branch.name}
                </p>
              )}
              <p>
                <strong>Created:</strong> {new Date(version.createdAt).toLocaleString()}
              </p>
              {version.updatedAt && (
                <p>
                  <strong>Updated:</strong> {new Date(version.updatedAt).toLocaleString()}
                </p>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div>
          <p>No versions found.</p>
          <p>Create a version using the CLI:</p>
          <code
            style={{ background: "#f0f0f0", padding: "10px", display: "block", marginTop: "10px" }}
          >
            ./bin/unkey create --workspace-id=ws_local_root --project-id=YOUR_PROJECT_ID
            --branch=main
          </code>
        </div>
      )}
    </div>
  );
}
