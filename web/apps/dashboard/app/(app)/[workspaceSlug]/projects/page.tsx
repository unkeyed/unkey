"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { type DeployCheckoutOrigin, routes } from "@/lib/navigation/routes";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef } from "react";
import { ProjectsPrototype } from "./_components/prototype/projects-prototype";

export default function ProjectsPage() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const searchParams = useSearchParams();
  const isNewProject = searchParams.get("new") === "true";

  usePendingSubscribe();

  // Legacy entry links (?new=true) used to open the create-project dialog
  // here; the flow now lives on its own full-screen page.
  useEffect(() => {
    if (isNewProject) {
      router.replace(routes.projects.new({ workspaceSlug: workspace.slug }));
    }
  }, [isNewProject, router, workspace.slug]);

  return <ProjectsPrototype />;
}

/**
 * Handles the Compute-plan gate hand-off: reads ?pendingPlan&from from the URL
 * and toasts the result, returning the user to where they came from on
 * `from=create`: the project's Apps page when projectId is present, otherwise
 * the full-page create-project flow.
 *
 * Two entry conditions land here, and the entitlement-first check absorbs both:
 * - Card on file (has-card path): the workspace is not yet subscribed, so
 *   subscribeDeploy runs here (no Stripe round-trip — the card is vaulted).
 * - Returning from a subscription-mode Compute checkout: /success (and the
 *   checkout.session.completed webhook) already linked the subscription, so the
 *   workspace is entitled and the entitlement check short-circuits to the
 *   toast/dialog with no subscribeDeploy call.
 *
 * subscribeDeploy and its BAD_REQUEST decline-recovery stay for the has-card
 * path and the setup-mode fallback (workspace already has a subscription, so it
 * vaults a card and attaches Compute items on return). Params are stripped
 * after capture so a refresh doesn't re-fire, and a ref guards double-firing.
 *
 * The params must be read reactively, not captured at mount: the has-card path
 * pushes ?pendingPlan&from while the user is ALREADY on the projects page (the
 * gate dialog lives here), so there is no remount — only a searchParams change.
 */
function usePendingSubscribe() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const trpcUtils = trpc.useUtils();

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
    // Set when the gate fired from a project's app-create flow; on success the
    // user is returned there instead of the create-project dialog.
    const projectId = searchParams.get("projectId") ?? undefined;
    const pending = { plan, from, projectId };
    const key = `${pending.plan}:${pending.from}:${pending.projectId ?? ""}`;
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
        if (pending.projectId) {
          // The gate fired from that project's app-create flow; land on the
          // project's Apps list (where "Create app" lives) rather than
          // dropping the user back into the wizard uninvited.
          router.push(
            routes.projects.apps.list({
              workspaceSlug: workspace.slug,
              projectId: pending.projectId,
            }),
          );
        } else {
          router.push(routes.projects.new({ workspaceSlug: workspace.slug }));
        }
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
                  projectId: pending.projectId,
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
}

const DEPLOY_ORIGINS: readonly DeployCheckoutOrigin[] = ["create", "banner", "billing"];

function planLabel(plan: string): string {
  return plan.charAt(0).toUpperCase() + plan.slice(1);
}
