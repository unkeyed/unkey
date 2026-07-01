"use client";

import { NewNavigationBanner } from "@/components/navigation/new-navigation-banner";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
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

  // Hook order: must run unconditionally, before the empty-state early return.
  const { createDialogOpen, setCreateDialogOpen } = usePendingSubscribe();

  if (!projects.isLoading && projects.data.length === 0) {
    return (
      <>
        <EmptyProjects />
        <CreateProjectDialog
          isOpen={createDialogOpen}
          onOpenChange={setCreateDialogOpen}
          workspaceSlug={workspace.slug}
        />
      </>
    );
  }

  return (
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
      <CreateProjectDialog
        isOpen={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        workspaceSlug={workspace.slug}
      />
    </PageContainer>
  );
}

/**
 * Handles the Compute-plan gate hand-off: reads ?pendingPlan&from from the URL
 * (set either by the has-card path or by /success after Stripe), subscribes the
 * plan, toasts the result, and on `from=create` opens the create-project
 * dialog. The card is on file by the time we get here, so subscribeDeploy needs
 * no Stripe round-trip. Params are stripped after capture so a refresh doesn't
 * re-fire, and a ref guards against double-firing within a render window.
 */
function usePendingSubscribe() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const trpcUtils = trpc.useUtils();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  const [pending] = useState(() => {
    const rawPlan = searchParams.get("pendingPlan");
    const plan = DEPLOY_PLANS.find((known) => known === rawPlan);
    if (!plan) {
      return null;
    }
    return { plan, fromCreate: searchParams.get("from") === "create" };
  });

  const fired = useRef(false);

  const subscribe = trpc.stripe.subscribeDeploy.useMutation();

  useEffect(() => {
    if (!pending || fired.current) {
      return;
    }
    fired.current = true;

    router.replace(routes.projects.list({ workspaceSlug: workspace.slug }));

    const markActive = async () => {
      toast.success(`${planLabel(pending.plan)} plan active`);
      await Promise.all([
        trpcUtils.stripe.getDeployEntitlement.invalidate(),
        trpcUtils.stripe.getDeploySubscription.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
      ]);
      if (pending.fromCreate) {
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
            // A precondition failure (e.g. no card on file) also won't clear on
            // retry; only offer Retry for transient/payment errors.
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
  }, [pending, router, workspace.slug, subscribe, trpcUtils]);

  return { createDialogOpen, setCreateDialogOpen };
}

function planLabel(plan: string): string {
  return plan.charAt(0).toUpperCase() + plan.slice(1);
}
