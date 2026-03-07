"use client";
import { trpc } from "@/lib/trpc/client";
import { Loading } from "@unkey/ui";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";

export default function AuthorizeDeploymentPage() {
  const params = useParams<{ workspaceSlug: string; projectId: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();

  const branch = searchParams.get("branch");

  const authorize = trpc.deploy.deployment.authorize.useMutation({
    onSuccess: () => {
      router.push(`/${params.workspaceSlug}/projects/${params.projectId}/deployments`);
    },
  });

  const [dismissed, setDismissed] = useState(false);

  if (!branch) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <h2 className="text-lg font-semibold text-content">Missing branch parameter</h2>
          <p className="text-content-subtle mt-2">
            This page should be accessed from a GitHub PR check link.
          </p>
        </div>
      </div>
    );
  }

  if (dismissed) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <h2 className="text-lg font-semibold text-content">Dismissed</h2>
          <p className="text-content-subtle mt-2">
            The deployment was not authorized. The GitHub check will remain as failed.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-[400px]">
      <div className="border border-border rounded-lg p-8 max-w-lg w-full">
        <h2 className="text-lg font-semibold text-content">Authorize Deployment</h2>
        <p className="text-content-subtle mt-2">
          An external contributor pushed to{" "}
          <code className="px-1.5 py-0.5 bg-background-subtle rounded text-sm font-mono">
            {branch}
          </code>
          . Authorize this deployment?
        </p>

        {authorize.error && (
          <div className="mt-4 p-3 bg-error-3 text-error-11 rounded-md text-sm">
            {authorize.error.message}
          </div>
        )}

        <div className="flex gap-3 mt-6">
          <button
            type="button"
            onClick={() =>
              authorize.mutate({
                projectId: params.projectId,
                branch,
              })
            }
            disabled={authorize.isLoading}
            className="flex-1 px-4 py-2 bg-accent-9 text-white rounded-md hover:bg-accent-10 disabled:opacity-50 font-medium text-sm"
          >
            {authorize.isLoading ? <Loading /> : "Approve"}
          </button>
          <button
            type="button"
            onClick={() => setDismissed(true)}
            disabled={authorize.isLoading}
            className="flex-1 px-4 py-2 border border-border rounded-md hover:bg-background-subtle text-content text-sm"
          >
            Dismiss
          </button>
        </div>
      </div>
    </div>
  );
}
