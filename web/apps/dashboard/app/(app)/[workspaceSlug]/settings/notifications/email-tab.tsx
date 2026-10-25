"use client";

import { Switch } from "@/components/ui/switch";
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
import { useResourceNames } from "./notification-ui";
import { useNotificationPreview } from "./preview-context";

export function EmailTab() {
  const preview = useNotificationPreview();
  const { scopeLabel } = useResourceNames();
  const me = preview.members.find((member) => member.userId === preview.currentUserId);

  if (!me) {
    return <p className="text-sm text-gray-11">Sign in to manage your email preferences.</p>;
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="max-w-xl space-y-2 text-sm text-gray-11">
        <p className="break-words">
          Email to <span className="text-gray-12">{me.email}</span>
        </p>
        <p id="email-preferences-description">
          On only allows email from enabled rules that include you. It does not subscribe you to an
          event. Off stops your email without changing shared destinations.
        </p>
      </div>

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
                const statusId = `${switchId}-status`;
                return (
                  <SettingsRow key={type.type}>
                    <SettingsRowHeader className="min-w-0">
                      <SettingsRowTitle>
                        <label htmlFor={switchId}>{type.label}</label>
                      </SettingsRowTitle>
                      <SettingsRowDescription id={descriptionId}>
                        {type.description}
                      </SettingsRowDescription>
                    </SettingsRowHeader>
                    <SettingsRowContent className="flex items-start justify-between gap-4">
                      <div
                        id={statusId}
                        className="min-w-0 flex-1 space-y-2 break-words text-xs leading-5 text-gray-11"
                      >
                        <p className="font-medium text-gray-12">
                          {enabled
                            ? routes.length > 0
                              ? "Email on for matching events"
                              : "Inactive"
                            : "Email off"}
                        </p>
                        {routes.length > 0 ? (
                          <ul className="space-y-2">
                            {routes.map(({ rule, via }) => (
                              <li key={rule.id}>
                                <p>{rule.name}</p>
                                <p>
                                  {via} ·{" "}
                                  {type.resourceScoped ? scopeLabel(rule.scope) : "Workspace-wide"}
                                </p>
                              </li>
                            ))}
                          </ul>
                        ) : (
                          <p>No enabled rules include you.</p>
                        )}
                      </div>
                      <Switch
                        id={switchId}
                        aria-describedby={`${descriptionId} ${statusId} email-preferences-description`}
                        checked={enabled}
                        onCheckedChange={(next) => {
                          preview.actions.setEmailPreference({
                            userId: me.userId,
                            notificationType: type.type,
                            enabled: next,
                          });
                          toast.success(
                            `${type.label} email preference ${next ? "on" : "off"} (preview)`,
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
