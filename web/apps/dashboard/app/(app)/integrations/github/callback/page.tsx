"use client";

export const dynamic = "force-dynamic";
export const revalidate = 0;

import { PageLoading } from "@/components/dashboard/page-loading";
import { trpc } from "@/lib/trpc/client";
import { Empty, toast } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo } from "react";

export default function Page() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const installationId = searchParams?.get("installation_id") ?? null;
  const state = searchParams?.get("state") ?? null;
  const installationIdNumber = useMemo(() => {
    if (!installationId) {
      return null;
    }

    const parsed = Number.parseInt(installationId, 10);
    return Number.isNaN(parsed) ? null : parsed;
  }, [installationId]);

  const mutation = trpc.github.registerInstallation.useMutation({
    onSuccess: (data) => {
      toast.success("GitHub App installed");
      router.push(`/${data.workspaceSlug}/projects/${data.projectId}/settings`);
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  useEffect(() => {
    if (!state || installationIdNumber === null) {
      return;
    }

    if (mutation.isIdle) {
      mutation.mutate({
        state,
        installationId: installationIdNumber,
      });
    }
  }, [mutation, state, installationIdNumber]);

  if (!state) {
    return (
      <div className="w-full h-screen flex justify-center items-center">
        <Empty>
          <Empty.Title>Invalid callback state</Empty.Title>
          <Empty.Description>
            Missing or invalid GitHub installation state.
          </Empty.Description>
        </Empty>
      </div>
    );
  }

  if (installationIdNumber === null) {
    return (
      <div className="w-full h-screen flex justify-center items-center">
        <Empty>
          <Empty.Title>Missing installation</Empty.Title>
          <Empty.Description>
            Missing or invalid GitHub installation id.
          </Empty.Description>
        </Empty>
      </div>
    );
  }

  if (mutation.isError) {
    return (
      <div className="w-full h-screen flex justify-center items-center">
        <Empty>
          <Empty.Title>Installation failed</Empty.Title>
          <Empty.Description>{mutation.error.message}</Empty.Description>
        </Empty>
      </div>
    );
  }

  return <PageLoading message="Finalizing GitHub installation..." />;
}
