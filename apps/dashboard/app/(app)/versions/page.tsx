"use client";

import { trpc } from "@/lib/trpc/client";

export default function DeploymentsPage() {
  const { data, isLoading, error } = trpc.deployment.list.useQuery();

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
        <p style={{ color: "red" }}>Failed to load deployments: {error.message}</p>
      </div>
    );
  }

  return (
    <div style={{ padding: "20px" }}>
      <h1>All Deployments</h1>

      {data?.deployments && data.deployments.length > 0 ? (
        <div>
          {data.deployments.map((deployment) => (
            <div
              key={deployment.id}
              style={{
                marginBottom: "20px",
                padding: "15px",
                border: "1px solid #ddd",
                borderRadius: "4px",
              }}
            >
              <h3>Deployment {deployment.id}</h3>
              <p>
                <strong>Status:</strong> {deployment.status}
              </p>
              {deployment.gitCommitSha && (
                <p>
                  <strong>Git Commit:</strong> {deployment.gitCommitSha}
                </p>
              )}
              {deployment.gitBranch && (
                <p>
                  <strong>Git Branch:</strong> {deployment.gitBranch}
                </p>
              )}
              {deployment.rootfsImageId && (
                <p>
                  <strong>Rootfs Image:</strong> {deployment.rootfsImageId}
                </p>
              )}
              {deployment.buildId && (
                <p>
                  <strong>Build ID:</strong> {deployment.buildId}
                </p>
              )}
              {deployment.project && (
                <p>
                  <strong>Project:</strong> {deployment.project.name} ({deployment.project.slug})
                </p>
              )}
              {deployment.branch && (
                <p>
                  <strong>Branch:</strong> {deployment.branch.name}
                </p>
              )}
              <p>
                <strong>Created:</strong> {new Date(deployment.createdAt).toLocaleString()}
              </p>
              {deployment.updatedAt && (
                <p>
                  <strong>Updated:</strong> {new Date(deployment.updatedAt).toLocaleString()}
                </p>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div>
          <p>No deployments found.</p>
          <p>Create a deployment using the CLI:</p>
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
