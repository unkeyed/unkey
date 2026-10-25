"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { IconArrowUpRightOutline12, IconPlusOutline12 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { useRuleEditor } from "./editors";
import { rulesCoveringResource } from "./notification-model";
import { findNotificationType } from "./notification-types";
import { PreviewBadge } from "./notification-ui";
import { NotificationPreviewProvider, useNotificationPreview } from "./preview-context";
import { RuleRow } from "./rules-tab";

export function ResourceNotifications(props: { projectId: string; appId?: string }) {
  return (
    <NotificationPreviewProvider>
      <ResourceNotificationsCard {...props} />
    </NotificationPreviewProvider>
  );
}

function ResourceNotificationsCard({
  projectId,
  appId,
}: {
  projectId: string;
  appId?: string;
}) {
  const workspace = useWorkspaceNavigation();
  const preview = useNotificationPreview();
  const ruleEditor = useRuleEditor();
  const resourceRules = preview.rules.filter(
    (rule) => findNotificationType(rule.notificationType)?.resourceScoped ?? false,
  );
  const { scoped, inherited } = rulesCoveringResource(resourceRules, { projectId, appId });
  const noun = appId ? "app" : "project";

  return (
    <section className="overflow-hidden rounded-lg border bg-raised">
      <div className="flex items-start justify-between gap-4 px-4 py-4">
        <div className="flex min-w-0 flex-col gap-1">
          <span className="flex items-center gap-2 text-sm font-medium text-gray-12">
            Notifications
            <PreviewBadge />
          </span>
          <span className="text-xs text-gray-9">
            Who hears about deployments in this {noun}. Rules are managed with the rest of your
            workspace notifications.
          </span>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            render={
              <Link href={routes.settings.notifications({ workspaceSlug: workspace.slug })} />
            }
          >
            All rules
            <IconArrowUpRightOutline12 />
          </Button>
          <Button
            variant="primary"
            size="sm"
            disabled={!preview.canManage}
            onClick={() =>
              ruleEditor.open({
                mode: "create",
                notificationType: "deployment.failed",
                scope: { projectId, appId: appId ?? null, environmentId: null },
              })
            }
          >
            <IconPlusOutline12 />
            Add rule for this {noun}
          </Button>
        </div>
      </div>

      <div className="divide-y divide-grayA-3 border-t border-grayA-4">
        {scoped.length === 0 ? (
          <p className="px-4 py-3 text-xs text-gray-9">No rules target this {noun} yet.</p>
        ) : (
          scoped.map((rule) => (
            <RuleRow
              key={rule.id}
              rule={rule}
              canManage={preview.canManage}
              onEdit={ruleEditor.open}
              showType
            />
          ))
        )}
      </div>

      {inherited.length > 0 ? (
        <div className="border-t border-grayA-4 bg-grayA-1 px-4 py-3 text-xs text-gray-9">
          {inherited.length} broader{" "}
          {inherited.length === 1 ? "rule also applies" : "rules also apply"}:{" "}
          {inherited.map((rule) => rule.name).join(", ")}
        </div>
      ) : null}

      {ruleEditor.element}
    </section>
  );
}
