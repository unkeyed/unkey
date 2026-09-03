"use client";

import { Switch } from "@/components/ui/switch";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { InfoTooltip, SettingCard, toast } from "@unkey/ui";

export function DeployAnomalyEmails() {
  const workspace = useWorkspaceNavigation();
  const { user } = useWorkspace();
  const utils = trpc.useUtils();
  const muted = workspace.betaFeatures.deploy_anomaly_alerts_muted ?? false;
  const isAdmin = user?.role === "admin";
  const update = trpc.workspace.updateDeployAnomalyEmails.useMutation({
    async onSuccess(result) {
      await utils.workspace.getCurrent.invalidate();
      toast.success(result.muted ? "Deploy anomaly emails muted" : "Deploy anomaly emails enabled");
    },
    onError(error) {
      toast.error("We couldn't update Deploy anomaly emails", { description: error.message });
    },
  });

  return (
    <SettingCard
      title="Deploy anomaly emails"
      description="Email workspace admins when production metrics deviate from their recent baseline. Alerts still appear in the dashboard when emails are muted."
      border="none"
      className="border-b border-grayA-4"
      contentWidth="w-full lg:w-[420px] justify-end"
    >
      <div className="flex w-full justify-end">
        <InfoTooltip
          content="Admin access required to change anomaly emails"
          disabled={isAdmin}
          asChild
        >
          <span>
            <Switch
              aria-label="Deploy anomaly emails"
              checked={!muted}
              disabled={!isAdmin || update.isLoading}
              onCheckedChange={(enabled) => update.mutate({ muted: !enabled })}
            />
          </span>
        </InfoTooltip>
      </div>
    </SettingCard>
  );
}
