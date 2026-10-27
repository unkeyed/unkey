"use client";

import { Switch } from "@/components/ui/switch";
import {
  useNotificationCatalog,
  useNotificationPreferences,
  useNotificationSession,
} from "@/lib/notifications/use-notifications";
import { getErrorMessage } from "@/lib/unkey-client";
import {
  Button,
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
import { NOTIFICATION_CATEGORIES, supportedNotificationTypes } from "./notification-types";

export function EmailTab() {
  const session = useNotificationSession();
  const catalog = useNotificationCatalog();
  const { query, mutation } = useNotificationPreferences();
  const me = session.user;

  if (session.isLoading) {
    return <output className="text-sm text-gray-11">Loading email preferences…</output>;
  }
  if (!me) {
    return <p className="text-sm text-gray-11">Sign in to manage your email preferences.</p>;
  }
  if (me.impersonator) {
    return (
      <p className="text-sm text-gray-11">Email preferences are unavailable while impersonating.</p>
    );
  }
  if (!query.data || !catalog.data) {
    const error = query.error ?? catalog.error;
    return (
      <div className="space-y-3 text-sm text-gray-11">
        <p role={error ? "alert" : "status"}>
          {error
            ? getErrorMessage(error, "Couldn't load email preferences.")
            : "Loading email preferences…"}
        </p>
        {error ? (
          <Button
            variant="outline"
            onClick={() => Promise.all([query.refetch(), catalog.refetch()])}
          >
            Retry
          </Button>
        ) : null}
      </div>
    );
  }
  const types = supportedNotificationTypes(catalog.data.eventTypes);

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
              {types
                .filter((type) => type.category === category)
                .map((type) => {
                  const preference = query.data.find(
                    (entry) => entry.notificationType === type.type,
                  );
                  const enabled = preference?.enabled ?? true;
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
                            {enabled ? "Email allowed for matching rules" : "Email off"}
                          </p>
                        </div>
                        <Switch
                          id={switchId}
                          aria-describedby={`${descriptionId} ${statusId} email-preferences-description`}
                          checked={enabled}
                          disabled={mutation.isLoading || !preference}
                          onCheckedChange={(next) => {
                            mutation.mutate(
                              { preferences: [{ notificationType: type.type, enabled: next }] },
                              {
                                onSuccess: () =>
                                  toast.success(
                                    `${type.label} email ${next ? "enabled" : "disabled"}`,
                                  ),
                              },
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
