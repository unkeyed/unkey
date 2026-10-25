"use client";

import { Switch } from "@/components/ui/switch";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsGroups,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
  toast,
} from "@unkey/ui";
import { emailRoutesForMember, isEmailEnabled } from "./notification-model";
import { NOTIFICATION_CATEGORIES, NOTIFICATION_TYPES } from "./notification-types";
import { useNotificationPreview } from "./preview-context";

export function EmailTab() {
  const preview = useNotificationPreview();
  const { workspace } = useWorkspace();
  const me = preview.members.find((member) => member.userId === preview.currentUserId);

  if (!me) {
    return <p className="text-sm text-gray-11">Sign in to manage your email preferences.</p>;
  }

  return (
    <div className="flex flex-col gap-6">
      <p className="max-w-xl text-sm text-gray-11">
        Manage your email at <span className="text-gray-12">{me.email}</span> for{" "}
        {workspace?.name ?? "this workspace"}. Workspace rules decide which events reach you.
        Turning an event off only stops your email; turning it on does not add you to any rule.
        Shared Slack, Discord, and webhook destinations are not affected.
      </p>

      <SettingsGroups className="max-w-none p-0">
        {NOTIFICATION_CATEGORIES.map(({ category, label }) => (
          <SettingsGroup key={category}>
            <SettingsGroupTitle>{label}</SettingsGroupTitle>
            <SettingsGroupContent>
              {NOTIFICATION_TYPES.filter((type) => type.category === category).map((type) => {
                const routes = emailRoutesForMember(preview, me.userId, type.type);
                const enabled = isEmailEnabled(preview.preferences, me.userId, type.type);
                const switchId = `email-${type.type}`;
                const descriptionId = `${switchId}-description`;
                return (
                  <SettingsRow key={type.type}>
                    <SettingsRowHeader>
                      <SettingsRowTitle>
                        <label htmlFor={switchId}>{type.label}</label>
                      </SettingsRowTitle>
                      <SettingsRowDescription id={descriptionId}>
                        <p>{type.description}</p>
                        <p className="mt-1">
                          {routes.length > 0
                            ? `Rules that include you: ${routes.map(({ rule, via }) => `${rule.name} (${via})`).join(", ")}`
                            : "No rule currently includes you."}
                        </p>
                      </SettingsRowDescription>
                    </SettingsRowHeader>
                    <SettingsRowContent className="flex items-center justify-end gap-3">
                      <span className="text-xs text-gray-11">
                        {enabled ? "Email allowed" : "Email off"}
                      </span>
                      <Switch
                        id={switchId}
                        aria-describedby={descriptionId}
                        checked={enabled}
                        onCheckedChange={(next) => {
                          preview.actions.setEmailPreference({
                            userId: me.userId,
                            notificationType: type.type,
                            enabled: next,
                          });
                          toast.success(
                            next
                              ? `Email allowed for ${type.label} (preview)`
                              : `Email turned off for ${type.label} (preview)`,
                          );
                        }}
                      />
                    </SettingsRowContent>
                  </SettingsRow>
                );
              })}
            </SettingsGroupContent>
          </SettingsGroup>
        ))}
      </SettingsGroups>
    </div>
  );
}
