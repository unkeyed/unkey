"use client";
import { LoadingState } from "@/components/loading-state";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Empty } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef } from "react";

export default function Page() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const installationId = searchParams?.get("installation_id") ?? null;
  const state = searchParams?.get("state") ?? null;
  // OAuth code GitHub returns when the App requests user authorization during
  // installation. The server uses it to verify the caller can access this
  // installation before binding it to their workspace.
  const code = searchParams?.get("code") ?? null;
  const installationIdNumber = useMemo(() => {
    if (!installationId) {
      return null;
    }

    const parsed = Number.parseInt(installationId, 10);
    return Number.isNaN(parsed) ? null : parsed;
  }, [installationId]);

  const mutation = trpc.github.registerInstallation.useMutation({
    onSuccess: (data) => {
      if (data.flow === "app" && data.projectId && data.appId) {
        // Return to the app: its settings, or the repo picker when the wizard
        // hasn't chosen a repo yet.
        router.replace(
          data.returnTo === "settings"
            ? routes.projects.apps.settings({
                workspaceSlug: data.workspaceSlug,
                projectId: data.projectId,
                appId: data.appId,
              })
            : routes.projects.apps.new({
                workspaceSlug: data.workspaceSlug,
                projectId: data.projectId,
                step: "select-repo",
                appId: data.appId,
              }),
        );
        return;
      }

      // "workspace" and "api" flows both land on workspace settings.
      router.replace(routes.settings.general({ workspaceSlug: data.workspaceSlug }));
    },
  });

  // OAuth code is single-use; fire once. A ref, not mutation.isIdle (which the
  // strict-mode remount reads before the first mutate flips it), blocks a re-submit.
  const submittedRef = useRef(false);
  useEffect(() => {
    if (!state || installationIdNumber === null || submittedRef.current) {
      return;
    }
    submittedRef.current = true;

    // `code` is absent when an existing user returns from editing an
    // already-authorized installation. The server only requires it when
    // binding an installation the workspace does not already own.
    mutation.mutate({
      state,
      installationId: installationIdNumber,
      code: code ?? undefined,
    });
  }, [mutation, state, installationIdNumber, code]);

  if (!state) {
    return (
      <div className="w-full min-h-[60vh] flex justify-center items-center">
        <Empty>
          <Empty.Title>Invalid callback state</Empty.Title>
          <Empty.Description>Missing or invalid GitHub installation state.</Empty.Description>
        </Empty>
      </div>
    );
  }

  if (installationIdNumber === null) {
    return (
      <div className="w-full min-h-[60vh] flex justify-center items-center">
        <Empty>
          <Empty.Title>Missing installation</Empty.Title>
          <Empty.Description>Missing or invalid GitHub installation id.</Empty.Description>
        </Empty>
      </div>
    );
  }

  if (mutation.isError) {
    return (
      <div className="w-full min-h-[60vh] flex justify-center items-center">
        <Empty>
          <Empty.Title>Installation failed</Empty.Title>
          <Empty.Description>{mutation.error.message}</Empty.Description>
        </Empty>
      </div>
    );
  }

  return <LoadingState message="Finalizing GitHub installation..." />;
}
