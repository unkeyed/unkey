"use client";

import { NewNavigationBanner } from "@/components/navigation/new-navigation-banner";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { type DeployCheckoutOrigin, routes } from "@/lib/navigation/routes";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { useLiveQuery } from "@tanstack/react-db";
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
  const projects = useLiveQuery((q) => q.from({ project: collection.projects }));

  const { createDialogOpen, setCreateDialogOpen } = usePendingSubscribe();

  const isEmpty = !projects.isLoading && projects.data.length === 0;

  return (
    <>
      {isEmpty ? (
        <EmptyProjects />
      ) : (
        <PageContainer>
          <PageHeader>
            <PageHeaderContent>
              <PageHeaderTitle>Projects</PageHeaderTitle>
            </PageHeaderContent>
            <PageHeaderActions>
              <CreateProjectButton defaultOpen={isNewProject} workspaceSlug={workspace.slug} />
            </PageHeaderActions>
          </PageHeader>
          <PageBody>
            <ProjectsList />
            <NewNavigationBanner />
          </PageBody>
        </PageContainer>
      )}
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
 * and toasts the result, opening the create-project dialog on `from=create`.
 *
 * The normal entry is a return from subscription-mode Compute checkout:
 * /success (and the checkout.session.completed webhook) already linked the
 * subscription, so the entitlement check short-circuits to the toast/dialog
 * with no subscribeDeploy call.
 *
 * subscribeDeploy remains only as the no-invoice resume path for an existing
 * same-tier subscription which is cancelling at period end. Old/bookmarked
 * pendingPlan URLs restart through Checkout. Params are stripped after capture
 * so a refresh doesn't re-fire, and a ref guards double-firing.
 *
 * The params must be read reactively rather than captured at mount because the
 * projects page can remain mounted across the checkout return navigation.
 */
function usePendingSubscribe() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const trpcUtils = trpc.useUtils();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);

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
    const from = DEPLOY_ORIGINS.find((known) => known === rawFrom) ?? "banner";
    const resume = searchParams.get("resume") === "true";
    const pending = { plan, from, resume };
    const key = `${pending.plan}:${pending.from}:${pending.resume}`;
    if (firedFor.current === key) {
      return;
    }
    firedFor.current = key;

    router.replace(routes.projects.list({ workspaceSlug: workspace.slug }));

    const markActive = async () => {
      toast.success(`${planLabel(pending.plan)} plan active`);
      await Promise.all([
        trpcUtils.stripe.getDeployEntitlement.invalidate(),
        trpcUtils.stripe.getDeploySubscription.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
      ]);
      if (pending.from === "create") {
        setCreateDialogOpen(true);
      }
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

    const resumeSubscription = () => {
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
            // Other preconditions won't clear on retry; surface the reason.
            if (error.data?.code === "PRECONDITION_FAILED") {
              toast.error(error.message || "Couldn't start your plan");
              return;
            }
            toast.error(error.message || "Couldn't start your plan", {
              action: { label: "Retry", onClick: resumeSubscription },
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
      if (pending.resume) {
        resumeSubscription();
        return;
      }
      router.push(
        routes.settings.stripe.checkout({
          workspaceSlug: workspace.slug,
          intent: "deploy",
          plan: pending.plan,
          from: pending.from,
        }),
      );
    })();
  }, [searchParams, router, workspace.slug, subscribe, trpcUtils]);

  return { createDialogOpen, setCreateDialogOpen };
}

const DEPLOY_ORIGINS: readonly DeployCheckoutOrigin[] = ["create", "banner", "billing"];

function planLabel(plan: string): string {
  return plan.charAt(0).toUpperCase() + plan.slice(1);
}
