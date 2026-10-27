"use client";

import { Switch } from "@/components/ui/switch";
import { notificationError } from "@/lib/notifications/use-notifications";
import { Collapsible } from "@base-ui/react/collapsible";
import { IconChevronDownOutline12, IconPlusOutline12 } from "@unkey/icons";
import {
  Badge,
  Button,
  FormSelect,
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  SettingsGroup,
  SettingsGroupTitle,
  cn,
  toast,
} from "@unkey/ui";
import { parseAsString, useQueryStates } from "nuqs";
import { useState } from "react";
import { useNotifications } from "./notification-context";
import { type ResourceScope, type Rule, rulesCoveringResource } from "./notification-model";
import {
  NOTIFICATION_CATEGORIES,
  type NotificationTypeDefinition,
  findNotificationType,
} from "./notification-types";
import { TargetChips, useResourceNames } from "./notification-ui";
import type { RuleEditorTarget } from "./rule-panel";

function RuleRow({
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
  const notifications = useNotifications();
  const [pending, setPending] = useState(false);
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
        disabled={!canManage || pending}
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
          destinations={notifications.destinations}
          members={notifications.members}
        />
      </button>
      <Switch
        size="sm"
        aria-label={`${rule.name} active`}
        disabled={!canManage || pending}
        aria-busy={pending}
        checked={rule.enabled}
        onCheckedChange={async (enabled) => {
          setPending(true);
          try {
            await notifications.actions.updateRule({ ...rule, enabled });
            toast.success(enabled ? `${rule.name} resumed` : `${rule.name} paused`);
          } catch (error) {
            notificationError(error, "Couldn't update rule");
          } finally {
            setPending(false);
          }
        }}
      />
    </ResourceListItem>
  );
}

function EventRow({
  type,
  rules,
  inherited,
  scope,
  onEdit,
}: {
  type: NotificationTypeDefinition;
  rules: Rule[];
  inherited: Rule[];
  scope: ResourceScope;
  onEdit: (target: RuleEditorTarget) => void;
}) {
  const notifications = useNotifications();
  const { scopeLabel } = useResourceNames();
  const ruleOnly = rules.length === 1 ? rules[0] : undefined;
  const addRoute = () => onEdit({ mode: "create", notificationType: type.type, scope });

  return (
    <Collapsible.Root render={<ResourceListItem />}>
      <div className="flex flex-wrap items-center gap-4 px-5 py-5 sm:flex-nowrap">
        <div className="min-w-0 basis-full sm:basis-2/5 sm:shrink-0">
          <Collapsible.Trigger
            disabled={rules.length === 0}
            aria-label={`Show routes for ${type.label}`}
            className="group flex items-center gap-2 rounded text-left text-sm font-medium text-gray-12 enabled:cursor-pointer focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-5"
          >
            {type.label}
            {rules.length > 0 ? (
              <IconChevronDownOutline12 className="shrink-0 text-gray-9 transition-transform group-data-[panel-open]:rotate-180" />
            ) : null}
          </Collapsible.Trigger>
          <p className="mt-1 text-xs leading-5 text-gray-11">{type.description}</p>
        </div>
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          {ruleOnly ? (
            <>
              <span className="text-xs text-gray-11">
                {type.resourceScoped ? scopeLabel(ruleOnly.scope) : "Whole workspace"}
                {ruleOnly.enabled ? null : " · Paused"}
              </span>
              <TargetChips
                targets={ruleOnly.targets}
                destinations={notifications.destinations}
                members={notifications.members}
              />
            </>
          ) : (
            <span className="text-xs text-gray-11">
              {rules.length > 0 ? `${rules.length} routes` : "No recipients"}
            </span>
          )}
        </div>
        {rules.length > 1 ? (
          <Collapsible.Trigger render={<Button variant="outline" size="sm" />}>
            Manage
          </Collapsible.Trigger>
        ) : (
          <Button
            variant="outline"
            size="sm"
            disabled={!notifications.canManage}
            aria-label={`${ruleOnly ? "Edit" : "Configure"} ${type.label}`}
            onClick={() => (ruleOnly ? onEdit({ mode: "edit", rule: ruleOnly }) : addRoute())}
          >
            {ruleOnly ? "Edit" : "Configure"}
          </Button>
        )}
      </div>
      <Collapsible.Panel className="border-t bg-grayA-2">
        <ResourceListBody>
          {rules.map((rule) => (
            <RuleRow
              key={rule.id}
              rule={rule}
              canManage={notifications.canManage}
              onEdit={onEdit}
              broader={inherited.includes(rule)}
            />
          ))}
        </ResourceListBody>
        <div className="px-4 py-3">
          <Button variant="ghost" size="sm" disabled={!notifications.canManage} onClick={addRoute}>
            <IconPlusOutline12 /> Add route
          </Button>
        </div>
      </Collapsible.Panel>
    </Collapsible.Root>
  );
}

export function EventRouting({ onEditRule }: { onEditRule: (target: RuleEditorTarget) => void }) {
  const notifications = useNotifications();
  const resources = useResourceNames();
  const [scope, setScope] = useQueryStates({
    projectId: parseAsString,
    appId: parseAsString,
    environmentId: parseAsString,
  });
  const [showFilters, setShowFilters] = useState(Boolean(scope.projectId));
  const project = resources.projects.find((candidate) => candidate.id === scope.projectId);
  const covering = scope.projectId
    ? rulesCoveringResource(
        notifications.rules.filter(
          (rule) => findNotificationType(rule.notificationType)?.resourceScoped,
        ),
        {
          projectId: scope.projectId,
          appId: scope.appId ?? undefined,
          environmentId: scope.environmentId ?? undefined,
        },
      )
    : { scoped: notifications.rules, inherited: [] };
  const rules = [...covering.scoped, ...covering.inherited];
  const ruleScope: ResourceScope = {
    projectId: scope.projectId,
    appId: scope.projectId ? scope.appId : null,
    environmentId: scope.appId ? scope.environmentId : null,
  };

  return (
    <ResourceList className="gap-6">
      <Collapsible.Root open={showFilters} onOpenChange={setShowFilters}>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-xs text-gray-11">
            Shared routes for your workspace. Each recipient gets one message.
          </p>
          <Collapsible.Trigger render={<Button variant="ghost" size="sm" />}>
            {scope.projectId ? resources.scopeLabel(ruleScope) : "Filter by resource"}
            <IconChevronDownOutline12 />
          </Collapsible.Trigger>
        </div>
        <Collapsible.Panel>
          {resources.isError ? (
            <p role="alert" className="text-xs text-error-11">
              Couldn't load resources. Refresh to try again.
            </p>
          ) : resources.isLoading ? (
            <output className="text-xs text-gray-11">Loading resources…</output>
          ) : null}
          <div className="mt-4 grid gap-3 sm:grid-cols-3">
            <FormSelect
              label="Project"
              disabled={resources.isLoading || resources.isError}
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
                ...resources.projects.map((project) => ({
                  value: project.id,
                  label: project.name,
                })),
              ]}
            />
            <FormSelect
              label="App"
              value={scope.appId ?? "all"}
              disabled={!project || resources.isLoading || resources.isError}
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
              disabled={!scope.appId || resources.isLoading || resources.isError}
              onValueChange={(value) => setScope({ environmentId: value === "all" ? null : value })}
              options={[
                { value: "all", label: "All environments" },
                ...resources.environments
                  .filter((environment) => environment.appId === scope.appId)
                  .map((environment) => ({ value: environment.id, label: environment.slug })),
              ]}
            />
          </div>
        </Collapsible.Panel>
      </Collapsible.Root>
      {scope.projectId ? (
        <p className="text-xs text-gray-11">
          Broader rules also apply here. Editing one changes it for every resource it covers.
        </p>
      ) : null}
      {notifications.canManage ? null : (
        <p className="text-xs text-gray-11">Only workspace admins can change rules.</p>
      )}
      {NOTIFICATION_CATEGORIES.filter(
        ({ category }) => !scope.projectId || category === "deployments",
      ).map(({ category, label }) => {
        const types = notifications.types.filter((type) => type.category === category);
        return (
          <SettingsGroup key={category}>
            <SettingsGroupTitle>{label}</SettingsGroupTitle>
            <ResourceListContent>
              <ResourceListBody>
                {types.map((type) => (
                  <EventRow
                    key={type.type}
                    type={type}
                    rules={rules.filter((rule) => rule.notificationType === type.type)}
                    inherited={covering.inherited}
                    scope={ruleScope}
                    onEdit={onEditRule}
                  />
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
