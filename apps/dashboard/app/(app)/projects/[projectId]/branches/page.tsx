"use client";

import { trpc } from "@/lib/trpc/client";
import { useParams } from "next/navigation";

export default function ProjectBranchesPage() {
  const params = useParams();
  const projectId = params?.projectId as string;

  const { data, isLoading, error } = trpc.project.branches.useQuery(
    {
      projectId,
    },
    {
      enabled: !!projectId, // Only run query if projectId exists
    },
  );

  if (!projectId) {
    return (
      <div style={{ padding: "20px" }}>
        <p>Invalid project ID</p>
      </div>
    );
  }

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
        <p style={{ color: "red" }}>Failed to load branches: {error.message}</p>
      </div>
    );
  }

  return (
    <div style={{ padding: "20px" }}>
      <h1>Branches for {data?.project.name}</h1>
      <p>
        Project: {data?.project.slug} ({data?.project.id})
      </p>

      <h2>All Branches</h2>
      {data?.branches && data.branches.length > 0 ? (
        <pre>{JSON.stringify(data.branches, null, 2)}</pre>
      ) : (
        <p>No branches found for this project.</p>
      )}
    </div>
  );
}
