"use client";
import { LoadingState } from "@/components/loading-state";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Github } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";

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

  // Captured from onSuccess. The not-granted branch doesn't navigate, and
  // reading mutation.data at render didn't reliably reflect the result here, so
  // we drive the recovery screen from state set in the callback.
  const [notGranted, setNotGranted] = useState<{
    repository: string;
    workspaceSlug: string;
    projectId: string;
    appId: string;
  } | null>(null);

  const mutation = trpc.github.registerInstallation.useMutation({
    onSuccess: (data) => {
      // Requested repo that wasn't granted during install: show the recovery
      // screen instead of navigating.
      if (data.requestedRepository && !data.repositoryConnected) {
        setNotGranted({
          repository: data.requestedRepository,
          workspaceSlug: data.workspaceSlug,
          projectId: data.projectId,
          appId: data.appId,
        });
        return;
      }

      if (data.returnTo === "settings" || data.repositoryConnected) {
        router.replace(
          routes.projects.apps.settings({
            workspaceSlug: data.workspaceSlug,
            projectId: data.projectId,
            appId: data.appId,
          }),
        );
      } else {
        router.replace(
          routes.projects.apps.new({
            workspaceSlug: data.workspaceSlug,
            projectId: data.projectId,
            step: "select-repo",
            appId: data.appId,
          }),
        );
      }
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

  if (notGranted) {
    return (
      <div className="w-full min-h-[60vh] flex justify-center items-center">
        <Empty>
          <Empty.Icon>
            <Github />
          </Empty.Icon>
          <div className="space-y-2.5">
            <Empty.Title>Repository not connected</Empty.Title>
            <Empty.Description className="max-w-150">
              <span className="text-gray-12 font-medium">{notGranted.repository}</span> isn't among the
              repositories you granted the app access to. Grant it on GitHub to finish connecting, or
              set it up later from the app's settings.
            </Empty.Description>
            <Empty.Actions>
              <Button
                className="px-3"
                variant="primary"
                onClick={() => {
                  // Reuse the current signed state so GitHub returns to this
                  // callback and the connection completes once the repo is granted.
                  window.location.href = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;
                }}
              >
                <Github className="size-4.5!" />
                Grant access on GitHub
              </Button>
              <Button
                className="px-3"
                variant="outline"
                onClick={() =>
                  router.replace(
                    routes.projects.apps.settings({
                      workspaceSlug: notGranted.workspaceSlug,
                      projectId: notGranted.projectId,
                      appId: notGranted.appId,
                    }),
                  )
                }
              >
                Go to settings
              </Button>
            </Empty.Actions>
          </div>
        </Empty>
      </div>
    );
  }

  return <LoadingState message="Finalizing GitHub installation..." />;
}
