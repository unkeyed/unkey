"use client";
import { CardPending } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/new/card";
import { PendingFlowCard } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/new/flow-layout";
import { LoadingState } from "@/components/loading-state";
import { signedState } from "@/lib/github-install-state";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import {
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  PageBody,
  PageContainer,
} from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { type ReactNode, useEffect, useMemo } from "react";

type PendingKind = "new-app" | "default";

function pendingKind(rawState: string | null): PendingKind {
  if (!rawState) {
    return "default";
  }
  let json: unknown;
  try {
    json = JSON.parse(rawState);
  } catch {
    return "default";
  }
  const parsed = signedState.safeParse(json);
  const newApp =
    parsed.success && parsed.data.flow === "app" && parsed.data.returnTo !== "settings";
  return newApp ? "new-app" : "default";
}

const pendingView: Record<PendingKind, ReactNode> = {
  "new-app": (
    <PendingFlowCard card="pick-repo">
      <CardPending message="Connecting GitHub…" />
    </PendingFlowCard>
  ),
  default: <LoadingState message="Finalizing GitHub installation..." />,
};

const submitted = new Set<string>();

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

  const utils = trpc.useUtils();
  const mutation = trpc.github.registerInstallation.useMutation({
    onSuccess: (data) => {
      if (data.status === "authorization_required") {
        window.location.replace(data.authorizationUrl);
        return;
      }

      if (data.flow === "app" && data.projectId && data.appId) {
        const scope = {
          workspaceSlug: data.workspaceSlug,
          projectId: data.projectId,
          appId: data.appId,
        };
        if (data.returnTo === "settings") {
          router.replace(routes.projects.apps.settings({ ...scope, page: "build" }));
          return;
        }
        void utils.github.listRepositories.prefetch({ projectId: data.projectId });
        router.replace(routes.projects.apps.new({ ...scope, step: "pick-repo" }));
        return;
      }

      // "workspace" and "api" flows both land on workspace settings.
      router.replace(routes.settings.general({ workspaceSlug: data.workspaceSlug }));
    },
  });

  // OAuth code is single-use; fire once. The set outlives a remount, which
  // mutation.isIdle and a ref do not, so it blocks a re-submit.
  useEffect(() => {
    const key = code ?? `${state}:${installationIdNumber}`;
    if (!state || (installationIdNumber === null && !code) || submitted.has(key)) {
      return;
    }
    submitted.add(key);

    // `code` is absent when GitHub returns from editing an existing
    // installation. The server starts an authorization round-trip if this
    // workspace has not linked the installation yet.
    mutation.mutate({
      state,
      installationId: installationIdNumber ?? undefined,
      code: code ?? undefined,
    });
  }, [mutation, state, installationIdNumber, code]);

  if (!state) {
    return (
      <CallbackError
        title="Invalid callback state"
        description="Missing or invalid GitHub installation state."
      />
    );
  }

  if (installationIdNumber === null && !code) {
    return (
      <CallbackError
        title="Missing installation"
        description="Missing or invalid GitHub installation id."
      />
    );
  }

  if (mutation.isError) {
    return <CallbackError title="Installation failed" description={mutation.error.message} />;
  }

  return pendingView[pendingKind(state)];
}

function CallbackError({ title, description }: { title: string; description: string }) {
  return (
    <PageContainer>
      <PageBody>
        <EmptyState>
          <EmptyStateHeader>
            <EmptyStateTitle>{title}</EmptyStateTitle>
            <EmptyStateDescription>{description}</EmptyStateDescription>
          </EmptyStateHeader>
        </EmptyState>
      </PageBody>
    </PageContainer>
  );
}
