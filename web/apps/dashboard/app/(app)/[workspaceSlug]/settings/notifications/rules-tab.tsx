"use client";

import { Switch } from "@/components/ui/switch";
import { IconPlusOutline12 } from "@unkey/icons";
import { Badge, Button, cn, toast } from "@unkey/ui";
import { type Rule, typeHasSendableRecipients } from "./notification-model";
import {
  NOTIFICATION_CATEGORIES,
  NOTIFICATION_TYPES,
  findNotificationType,
} from "./notification-types";
import { TargetChips, useResourceNames } from "./notification-ui";
import { useNotificationPreview } from "./preview-context";
import type { RuleEditorTarget } from "./rule-panel";

export function RuleRow({
  rule,
  canManage,
  onEdit,
  showType = false,
}: {
  rule: Rule;
  canManage: boolean;
  onEdit: (target: RuleEditorTarget) => void;
  showType?: boolean;
}) {
  const preview = useNotificationPreview();
  const { scopeLabel } = useResourceNames();

  return (
    <div className={cn("flex items-center gap-3 px-4 py-3", !rule.enabled && "opacity-60")}>
      <button
        type="button"
        disabled={!canManage}
        onClick={() => onEdit({ mode: "edit", rule })}
        aria-label={`Edit ${rule.name}`}
        className="group flex min-w-0 flex-1 flex-col gap-1.5 rounded-md text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7 enabled:cursor-pointer"
      >
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate group-hover:underline text-sm text-gray-12">{rule.name}</span>
          <span className="shrink-0 text-xs text-gray-9">
            {showType ? `${rule.notificationType} · ` : ""}
            {findNotificationType(rule.notificationType)?.resourceScoped
              ? scopeLabel(rule.scope)
              : "Whole workspace"}
          </span>
          {rule.enabled ? null : <Badge variant="secondary">Paused</Badge>}
        </span>
        <TargetChips
          targets={rule.targets}
          destinations={preview.destinations}
          members={preview.members}
        />
      </button>
      <Switch
        size="sm"
        aria-label={`${rule.name} active`}
        disabled={!canManage}
        checked={rule.enabled}
        onCheckedChange={(enabled) => {
          preview.actions.setRuleEnabled(rule.id, enabled);
          toast.success(
            enabled ? `${rule.name} resumed in preview` : `${rule.name} paused in preview`,
          );
        }}
      />
    </div>
  );
}

export function RulesTab({
  onEditRule,
  onTestRouting,
}: {
  onEditRule: (target: RuleEditorTarget) => void;
  onTestRouting: () => void;
}) {
  const preview = useNotificationPreview();

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4">
        <p className="max-w-xl text-sm text-gray-11">
          Rules choose who hears about each event. Rules add up: when several match, each person and
          destination still gets one message.
        </p>
        <div className="flex shrink-0 items-center gap-2">
          <Button variant="outline" onClick={onTestRouting}>
            Test routing
          </Button>
          <Button
            variant="primary"
            disabled={!preview.canManage}
            onClick={() => onEditRule({ mode: "create" })}
          >
            <IconPlusOutline12 />
            Add rule
          </Button>
        </div>
      </div>

      {preview.canManage ? null : (
        <p className="text-xs text-gray-9">Only workspace admins can change rules.</p>
      )}

      {NOTIFICATION_CATEGORIES.map(({ category, label }) => (
        <section key={category} className="flex flex-col gap-2">
          <h2 className="text-xs font-medium uppercase tracking-wide text-gray-9">{label}</h2>
          <div className="divide-y divide-grayA-4 overflow-hidden rounded-lg border bg-raised">
            {NOTIFICATION_TYPES.filter((type) => type.category === category).map((type) => {
              const rules = preview.rules.filter((rule) => rule.notificationType === type.type);
              const active = typeHasSendableRecipients(preview, type.type);
              return (
                <div key={type.type}>
                  <div className="flex items-center gap-3 bg-grayA-1 px-4 py-3">
                    <div className="flex min-w-0 flex-1 flex-col">
                      <span className="flex items-center gap-2">
                        <span className="text-sm font-medium text-gray-12">{type.label}</span>
                        <code className="text-xs text-gray-9">{type.type}</code>
                      </span>
                      <span className="text-xs text-gray-9">{type.description}</span>
                    </div>
                    {active ? null : <Badge variant="warning">Nobody is notified</Badge>}
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={!preview.canManage}
                      onClick={() => onEditRule({ mode: "create", notificationType: type.type })}
                      aria-label={`Add rule for ${type.label}`}
                    >
                      <IconPlusOutline12 />
                      Rule
                    </Button>
                  </div>
                  {rules.length > 0 ? (
                    <div className="divide-y divide-grayA-3 border-t border-grayA-3 pl-4">
                      {rules.map((rule) => (
                        <RuleRow
                          key={rule.id}
                          rule={rule}
                          canManage={preview.canManage}
                          onEdit={onEditRule}
                        />
                      ))}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        </section>
      ))}
    </div>
  );
}
