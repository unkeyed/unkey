"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useGuardedNavigate } from "@/components/leave-guard";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  dismissRedeployBanner,
  showRedeployBanner,
  useRedeployBannerVisible,
  useRedeployScope,
} from "@/lib/collections/deploy/pending-redeploy";
import { type RedeployTarget, useDeployActions, useRedeployTargets } from "@/lib/deploy-data";
import { routes } from "@/lib/navigation/routes";
import { ENVIRONMENT_KIND } from "@/lib/queries/environments";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { IconHammer2Outline18, IconXmarkOutline18 } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import { cn } from "cn";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { GlowIcon } from "../components/glow-icon";

const environmentList = new Intl.ListFormat("en", { type: "conjunction" });

export function PendingRedeployBanner() {
  const { refetchDeployments, refetchApps } = useDeployActions();
  const navigate = useGuardedNavigate();
  const workspace = useWorkspaceNavigation();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const visible = useRedeployBannerVisible();
  const scope = useRedeployScope();
  const targets = useRedeployTargets(visible ? scope : null) ?? [];
  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);

  const show = visible && targets.length > 0;

  const redeploy = useMutation({
    mutationFn: (targets: RedeployTarget[]) =>
      Promise.allSettled(
        targets.map(async ({ deployment }) => {
          const res = await getUnkeyClient().deployments.createDeploymentV3({
            project: deployment.projectId,
            app: deployment.appId,
            environment: deployment.environmentId,
            deployment: { deploymentId: deployment.id },
          });
          return res.data.deploymentId;
        }),
      ),
    onSuccess: async (results, targets) => {
      const failure = results.find(
        (result): result is PromiseRejectedResult => result.status === "rejected",
      );
      if (failure) {
        const failed = targets.filter((_, i) => results[i].status === "rejected");
        toast.error(
          `Redeploy failed for ${environmentList.format(failed.map(({ environment }) => environment.slug))}`,
          { description: getErrorMessage(failure.reason) },
        );
        showRedeployBanner(failed.map(({ environment }) => environment.id));
      } else {
        dismissRedeployBanner();
      }
      const started = targets.flatMap((target, i) => {
        const result = results[i];
        return result.status === "fulfilled" ? [{ target, deploymentId: result.value }] : [];
      });
      const opened =
        started.find(({ target }) => target.environment.kind === ENVIRONMENT_KIND.production) ??
        started[0];
      if (!opened) {
        return;
      }
      refetchDeployments();
      await refetchApps();
      navigate(
        routes.projects.apps.deployment({
          workspaceSlug: workspace.slug,
          projectId: opened.target.deployment.projectId,
          appId: opened.target.deployment.appId,
          deploymentId: opened.deploymentId,
        }),
      );
    },
  });

  useEffect(
    function getDismissedAutomatically() {
      if (!show) {
        return;
      }
      const timer = setTimeout(() => dismissRedeployBanner(scope), 10_000);
      return () => {
        clearTimeout(timer);
      };
    },
    [show, scope],
  );

  if (!mounted) {
    return null;
  }

  // The app shell isolates page content in its own stacking context, under the sticky top nav.
  return createPortal(
    <div
      aria-hidden={!show}
      inert={!show || undefined}
      className={cn(
        "fixed top-6 right-6 z-50 transition-[transform,opacity] duration-300 ease-out",
        show ? "translate-x-0 opacity-100" : "translate-x-[calc(100%+24px)] opacity-0",
      )}
    >
      <div className="relative flex items-start gap-4 rounded-xl bg-raised p-4 shadow-floating w-100">
        <button
          type="button"
          onClick={() => dismissRedeployBanner()}
          className="absolute top-3 right-3 text-gray-9 hover:text-gray-11 transition-colors cursor-pointer"
          aria-label="Dismiss"
        >
          <IconXmarkOutline18 className="size-4" />
        </button>

        <GlowIcon
          icon={<IconHammer2Outline18 className="size-4.5" />}
          className="w-9 h-9 shrink-0"
        />

        <div className="flex flex-col gap-3 flex-1 min-w-0">
          <div className="flex flex-col gap-1 pr-5">
            <span className="text-sm font-semibold text-gray-12 leading-5">Changes detected</span>
            <span className="text-xs text-gray-11 leading-4">
              Redeploy to apply your latest changes to{" "}
              {environmentList.format(targets.map(({ environment }) => environment.slug))}.
            </span>
          </div>
          <Button
            variant="primary"
            size="md"
            className="w-full"
            disabled={redeploy.isLoading}
            loading={redeploy.isLoading}
            onClick={() => {
              if (gated) {
                openPaywall();
                return;
              }
              redeploy.mutate(targets);
            }}
          >
            Redeploy
          </Button>
        </div>
      </div>
      {planGate}
    </div>,
    document.body,
  );
}
