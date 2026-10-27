"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import {
  useNotificationRules,
  useNotificationSession,
} from "@/lib/notifications/use-notifications";
import { getErrorMessage } from "@/lib/unkey-client";
import { IconArrowUpRightOutline12 } from "@unkey/icons";
import {
  Button,
  SettingsGroup,
  SettingsGroupContent,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import Link from "next/link";
import { rulesCoveringResource, toRuleModel } from "./notification-model";
import { findNotificationType } from "./notification-types";

type Resource = { projectId: string; appId?: string; environmentId?: string };

export function ResourceNotifications(resource: Resource) {
  const workspace = useWorkspaceNavigation();
  const session = useNotificationSession();
  const rules = useNotificationRules();
  if (!session.canManage) {
    return null;
  }
  const { scoped, inherited } = rulesCoveringResource(
    (rules.data ?? [])
      .map(toRuleModel)
      .filter((rule) => findNotificationType(rule.notificationType)?.resourceScoped),
    resource,
  );
  const active = [...scoped, ...inherited].filter((rule) => rule.enabled).length;
  const broader = inherited.filter((rule) => rule.enabled).length;
  const noun = resource.environmentId ? "environment" : resource.appId ? "app" : "project";

  return (
    <SettingsGroup>
      <SettingsGroupContent>
        <SettingsRow>
          <SettingsRowHeader>
            <SettingsRowTitle>Notifications</SettingsRowTitle>
            <SettingsRowDescription>
              Choose who hears about deployments in this {noun}. Shared workspace rules also apply.
            </SettingsRowDescription>
          </SettingsRowHeader>
          <SettingsRowContent>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex flex-col gap-1">
                {rules.data ? (
                  <>
                    <span className="text-sm text-gray-12">
                      {active} active {active === 1 ? "rule" : "rules"}
                    </span>
                    <span className="text-xs text-gray-11">
                      {broader > 0 ? `${broader} from broader scopes` : "No broader rules apply"}
                    </span>
                  </>
                ) : (
                  <span role={rules.error ? "alert" : "status"} className="text-sm text-gray-11">
                    {rules.error
                      ? getErrorMessage(rules.error, "Couldn't load rules.")
                      : "Loading rules…"}
                  </span>
                )}
              </div>
              <Button
                variant="outline"
                size="sm"
                render={
                  <Link
                    href={routes.settings.notifications({
                      workspaceSlug: workspace.slug,
                      ...resource,
                    })}
                  />
                }
              >
                Manage rules <IconArrowUpRightOutline12 />
              </Button>
            </div>
          </SettingsRowContent>
        </SettingsRow>
      </SettingsGroupContent>
    </SettingsGroup>
  );
}
