"use client";

import { Switch } from "@/components/ui/switch";
import { useWorkspace } from "@/providers/workspace-provider";
import { toast } from "@unkey/ui";
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
        Choose which emails you get from {workspace?.name ?? "this workspace"} at{" "}
        <span className="text-gray-12">{me.email}</span>. Admins decide which events reach you;
        turning one off here only stops your email. Slack, Discord, and webhook destinations are not
        affected.
      </p>

      {NOTIFICATION_CATEGORIES.map(({ category, label }) => (
        <section key={category} className="flex flex-col gap-2">
          <h2 className="text-xs font-medium uppercase tracking-wide text-gray-9">{label}</h2>
          <ul className="divide-y divide-grayA-4 overflow-hidden rounded-lg border bg-raised">
            {NOTIFICATION_TYPES.filter((type) => type.category === category).map((type) => {
              const routes = emailRoutesForMember(preview, me.userId, type.type);
              const enabled = isEmailEnabled(preview.preferences, me.userId, type.type);
              const switchId = `email-${type.type}`;
              return (
                <li key={type.type} className="flex items-center gap-4 px-4 py-3">
                  <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <label htmlFor={switchId} className="text-sm text-gray-12">
                      {type.label}
                    </label>
                    <span className="text-xs text-gray-9">{type.description}</span>
                    <span
                      className={routes.length > 0 ? "text-xs text-gray-11" : "text-xs text-gray-9"}
                    >
                      {routes.length > 0
                        ? `Sent to you by ${routes.map(({ rule, via }) => `${rule.name} (${via})`).join(", ")}`
                        : "No rule sends this to you right now."}
                    </span>
                  </div>
                  <span className="shrink-0 text-xs text-gray-9">
                    {enabled ? "Email on" : "Email off"}
                  </span>
                  <Switch
                    id={switchId}
                    checked={enabled}
                    onCheckedChange={(next) => {
                      preview.actions.setEmailPreference({
                        userId: me.userId,
                        notificationType: type.type,
                        enabled: next,
                      });
                      toast.success(
                        next
                          ? `Email turned on for ${type.label} (preview)`
                          : `Email turned off for ${type.label} (preview)`,
                      );
                    }}
                  />
                </li>
              );
            })}
          </ul>
        </section>
      ))}
    </div>
  );
}
