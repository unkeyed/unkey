"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import {
  dismissSettingsBanner,
  useSettingsBannerTarget,
  useSettingsBannerVisible,
} from "@/lib/collections/deploy/pending-redeploy";
import { IconHammer2Outline18, IconXmarkOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "cn";
import { useEffect } from "react";
import { useAppCurrentDeployment } from "../(overview)/hooks/use-app-current-deployment";
import { useRedeployDeployment } from "../(overview)/hooks/use-redeploy-deployment";
import { GlowIcon } from "../components/glow-icon";

export function PendingRedeployBanner() {
  const { app, currentDeployment } = useAppCurrentDeployment();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const visible = useSettingsBannerVisible();
  const target = useSettingsBannerTarget();
  const deployment = target ? target.deployment : currentDeployment;

  const show = visible && !!deployment && deployment.appId === app?.id;

  const redeploy = useRedeployDeployment();

  useEffect(
    function getDismissedAutomatically() {
      if (!show) {
        return;
      }
      const timer = setTimeout(() => dismissSettingsBanner(target), 10_000);
      return () => {
        clearTimeout(timer);
      };
    },
    [show, target],
  );

  return (
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
          onClick={() => dismissSettingsBanner()}
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
              Redeploy to apply your latest changes to {target?.environmentSlug ?? "production"}.
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
              if (deployment) {
                redeploy.mutate(deployment);
              }
            }}
          >
            Redeploy
          </Button>
        </div>
      </div>
      {planGate}
    </div>
  );
}
