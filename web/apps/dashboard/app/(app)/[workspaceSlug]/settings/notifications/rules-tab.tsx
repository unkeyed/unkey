"use client";

import { Switch } from "@/components/ui/switch";
import { IconPlusOutline12 } from "@unkey/icons";
import {
  Badge,
  Button,
  FormSelect,
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListHeader,
  ResourceListItem,
  SettingsGroup,
  SettingsGroupTitle,
  cn,
  toast,
} from "@unkey/ui";
import { parseAsString, useQueryStates } from "nuqs";
import {
  type ResourceScope,
  type Rule,
  rulesCoveringResource,
  typeHasSendableRecipients,
} from "./notification-model";
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
  broader = false,
}: {
  rule: Rule;
  canManage: boolean;
  onEdit: (target: RuleEditorTarget) => void;
  broader?: boolean;
}) {
  const preview = useNotificationPreview();
  const { scopeLabel } = useResourceNames();
  return (
    <ResourceListItem
      className={cn(
        "flex items-center gap-4 px-4 py-4 transition-colors hover:bg-grayA-2",
        !rule.enabled && "opacity-60",
      )}
    >
      <button
        type="button"
        disabled={!canManage}
        onClick={() => onEdit({ mode: "edit", rule })}
        aria-label={`Edit ${rule.name}`}
        className="group flex min-w-0 flex-1 flex-col gap-2 rounded-md text-left focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-5 enabled:cursor-pointer"
      >
        <span className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium text-gray-12 group-hover:underline">
            {rule.name}
          </span>
          {rule.enabled ? null : <Badge variant="secondary">Paused</Badge>}
          {broader ? <Badge variant="secondary">Broader rule</Badge> : null}
        </span>
        <span className="text-xs text-gray-11">
          {findNotificationType(rule.notificationType)?.label ?? rule.notificationType}
          {" · "}
          {findNotificationType(rule.notificationType)?.resourceScoped
            ? scopeLabel(rule.scope)
            : "Whole workspace"}
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
    </ResourceListItem>
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
  const resources = useResourceNames();
  const [scope, setScope] = useQueryStates({
    projectId: parseAsString,
    appId: parseAsString,
    environmentId: parseAsString,
  });
  const project = resources.projects.find((candidate) => candidate.id === scope.projectId);
  const covering = scope.projectId
    ? rulesCoveringResource(
        preview.rules.filter((rule) => findNotificationType(rule.notificationType)?.resourceScoped),
        {
          projectId: scope.projectId,
          appId: scope.appId ?? undefined,
          environmentId: scope.environmentId ?? undefined,
        },
      )
    : { scoped: preview.rules, inherited: [] };
  const rules = [...covering.scoped, ...covering.inherited];
  const ruleScope: ResourceScope = {
    projectId: scope.projectId,
    appId: scope.projectId ? scope.appId : null,
    environmentId: scope.appId ? scope.environmentId : null,
  };

  return (
    <ResourceList className="gap-6">
      <ResourceListHeader className="items-start justify-between gap-3">
        <p className="max-w-md text-sm leading-5 text-gray-11">
          Match an event to people and destinations. Matching rules add up; each recipient gets one
          message.
        </p>
        <div className="flex shrink-0 items-center gap-2">
          <Button variant="outline" size="sm" onClick={onTestRouting}>
            Preview recipients
          </Button>
          <Button
            variant="primary"
            size="sm"
            disabled={!preview.canManage}
            onClick={() => onEditRule({ mode: "create", scope: ruleScope })}
          >
            <IconPlusOutline12 />
            Add rule
          </Button>
        </div>
      </ResourceListHeader>
      <div className="grid gap-3 sm:grid-cols-3">
        <FormSelect
          label="Project"
          value={scope.projectId ?? "all"}
          onValueChange={(value) =>
            setScope({
              projectId: value === "all" ? null : value,
              appId: null,
              environmentId: null,
            })
          }
          options={[
            { value: "all", label: "All projects" },
            ...resources.projects.map((project) => ({ value: project.id, label: project.name })),
          ]}
        />
        <FormSelect
          label="App"
          value={scope.appId ?? "all"}
          disabled={!project}
          onValueChange={(value) =>
            setScope({ appId: value === "all" ? null : value, environmentId: null })
          }
          options={[
            { value: "all", label: "All apps" },
            ...(project?.apps ?? []).map((app) => ({ value: app.id, label: app.name })),
          ]}
        />
        <FormSelect
          label="Environment"
          value={scope.environmentId ?? "all"}
          disabled={!scope.appId}
          onValueChange={(value) => setScope({ environmentId: value === "all" ? null : value })}
          options={[
            { value: "all", label: "All environments" },
            ...resources.environments
              .filter((environment) => environment.appId === scope.appId)
              .map((environment) => ({ value: environment.id, label: environment.slug })),
          ]}
        />
      </div>
      {scope.projectId ? (
        <p className="text-xs text-gray-11">
          Broader rules also apply here. Editing one changes it for every resource it covers.
        </p>
      ) : null}
      {preview.canManage ? null : (
        <p className="text-xs text-gray-11">Only workspace admins can change rules.</p>
      )}
      {NOTIFICATION_CATEGORIES.filter(
        ({ category }) => !scope.projectId || category === "deployments",
      ).map(({ category, label }) => {
        const types = NOTIFICATION_TYPES.filter((type) => type.category === category);
        const categoryRules = rules.filter((rule) =>
          types.some((type) => type.type === rule.notificationType),
        );
        return (
          <SettingsGroup key={category}>
            <SettingsGroupTitle>{label}</SettingsGroupTitle>
            <ResourceListContent>
              <ResourceListBody>
                {categoryRules.map((rule) => (
                  <RuleRow
                    key={rule.id}
                    rule={rule}
                    canManage={preview.canManage}
                    onEdit={onEditRule}
                    broader={covering.inherited.includes(rule)}
                  />
                ))}
                {types
                  .filter((type) => !typeHasSendableRecipients({ ...preview, rules }, type.type))
                  .map((type) => (
                    <ResourceListItem
                      key={type.type}
                      className="flex items-center justify-between gap-3 px-4 py-4"
                    >
                      <div className="flex flex-col gap-1">
                        <span className="text-sm text-gray-12">{type.label}</span>
                        <span className="text-xs text-gray-11">
                          No active recipients{scope.projectId ? " in this scope" : ""}.
                        </span>
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={!preview.canManage}
                        onClick={() =>
                          onEditRule({
                            mode: "create",
                            notificationType: type.type,
                            scope: ruleScope,
                          })
                        }
                      >
                        <IconPlusOutline12 />
                        Add rule
                      </Button>
                    </ResourceListItem>
                  ))}
              </ResourceListBody>
            </ResourceListContent>
          </SettingsGroup>
        );
      })}
      {scope.projectId ? null : (
        <p className="text-xs text-gray-11">
          All projects includes projects and apps created later.
        </p>
      )}
    </ResourceList>
  );
}
