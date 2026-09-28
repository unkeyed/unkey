"use client";

import { ComputeUpgradeCelebration } from "@/components/billing/upgrade-success/upgrade-success-dialog";
import { useVisibleProjects } from "@/hooks/use-visible-projects";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { DEPLOY_CHECKOUT_ORIGINS, routes } from "@/lib/navigation/routes";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  toast,
} from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { CreateProjectButton } from "./_components/create-project-button";
import { CreateProjectDialog } from "./_components/create-project-dialog";
import { ProjectsList } from "./_components/list";
import { EmptyProjects } from "./_components/list/empty-projects";

export default function ProjectsPage() {
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const isNewProject = searchParams.get("new") === "true";
  const projects = useVisibleProjects();

  const { createDialogOpen, setCreateDialogOpen, welcome, closeWelcome } = usePendingSubscribe();

  const isEmpty = !projects.isLoading && projects.data.length === 0;

  return (
    <>
      <PageContainer>
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>Projects</PageHeaderTitle>
          </PageHeaderContent>
          <PageHeaderActions>
            <CreateProjectButton defaultOpen={isNewProject} workspaceSlug={workspace.slug} />
          </PageHeaderActions>
        </PageHeader>
        <PageBody>{isEmpty ? <EmptyProjects /> : <ProjectsList />}</PageBody>
      </PageContainer>
      {welcome ? <ComputeUpgradeCelebration plan={welcome.plan} onClose={closeWelcome} /> : null}
      <CreateProjectDialog
        isOpen={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        workspaceSlug={workspace.slug}
      />
    </>
  );
}

/**
 * Handles the Compute-plan gate hand-off: reads ?pendingPlan&from from the URL
 * and shows the welcome dialog, then the create-project dialog on `from=create`.
 *
 * The setup-mode checkout fallback (the workspace already had a subscription)
 * only vaults a card, so the workspace is not yet entitled and subscribeDeploy
 * must still run here.
 */
function usePendingSubscribe() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const trpcUtils = trpc.useUtils();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [welcome, setWelcome] = useState<{ plan: DeployPlan; thenCreate: boolean } | null>(null);
  const closeWelcome = () => {
    setWelcome(null);
    if (welcome?.thenCreate) {
      setCreateDialogOpen(true);
    }
  };

  // The pendingPlan+from pair currently being subscribed, so re-renders (and
  // strict-mode double effects) don't re-fire it. Cleared when the params are
  // gone so a later hand-off (subscribe → cancel → subscribe again) runs fresh.
  const firedFor = useRef<string | null>(null);

  const subscribe = trpc.stripe.subscribeDeploy.useMutation();

  useEffect(() => {
    const rawPlan = searchParams.get("pendingPlan");
    const plan = DEPLOY_PLANS.find((known) => known === rawPlan);
    if (!plan) {
      firedFor.current = null;
      return;
    }
    // Carry the raw origin (not a boolean) so a card-decline retry preserves
    // it verbatim instead of rewriting e.g. "billing" to "banner".
    const rawFrom = searchParams.get("from");
    const from = DEPLOY_CHECKOUT_ORIGINS.find((known) => known === rawFrom) ?? "banner";
    const pending = { plan, from };
    const key = `${pending.plan}:${pending.from}`;
    if (firedFor.current === key) {
      return;
    }
    firedFor.current = key;

    router.replace(routes.projects.list({ workspaceSlug: workspace.slug }));

    const markActive = async () => {
      await Promise.all([
        trpcUtils.stripe.getDeployEntitlement.invalidate(),
        trpcUtils.stripe.getDeploySubscription.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
      ]);
      setWelcome({ plan: pending.plan, thenCreate: pending.from === "create" });
    };

    // Re-entering this URL (bookmark, reshare, history remount) or a race can
    // hit a workspace that already has the plan, where subscribeDeploy throws
    // "already has a plan". Reading entitlement first lets us treat that as the
    // success it is instead of surfacing a scary error.
    const isEntitled = async () => {
      const entitlement = await trpcUtils.stripe.getDeployEntitlement
        .fetch(undefined, { staleTime: 0 })
        .catch(() => null);
      return Boolean(entitlement?.entitled);
    };

    const attempt = () => {
      subscribe.mutate(
        { plan: pending.plan },
        {
          onSuccess: markActive,
          onError: async (error) => {
            if (await isEntitled()) {
              await markActive();
              return;
            }
            // Non-admins are blocked server-side by requireWorkspaceAdmin; retry
            // can never clear it, so surface the reason without a Retry action.
            if (error.data?.code === "FORBIDDEN") {
              toast.error("Only workspace admins can manage billing.");
              return;
            }
            // Payment failure: the workspace has a Stripe customer but no usable
            // card, so the charge fails. Send them to Stripe to add one — /success
            // returns to this landing and re-subscribes.
            if (error.data?.code === "BAD_REQUEST") {
              router.push(
                routes.settings.stripe.checkout({
                  workspaceSlug: workspace.slug,
                  intent: "deploy",
                  plan: pending.plan,
                  from: pending.from,
                }),
              );
              return;
            }
            // Other preconditions won't clear on retry; surface the reason.
            if (error.data?.code === "PRECONDITION_FAILED") {
              toast.error(error.message || "Couldn't start your plan");
              return;
            }
            toast.error(error.message || "Couldn't start your plan", {
              action: { label: "Retry", onClick: attempt },
            });
          },
        },
      );
    };

    void (async () => {
      if (await isEntitled()) {
        await markActive();
        return;
      }
      attempt();
    })();
  }, [searchParams, router, workspace.slug, subscribe, trpcUtils]);

  return { createDialogOpen, setCreateDialogOpen, welcome, closeWelcome };
}
