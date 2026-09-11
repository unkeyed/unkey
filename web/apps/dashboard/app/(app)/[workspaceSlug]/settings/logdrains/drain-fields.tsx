"use client";

import {
  Multibox,
  MultiboxChip,
  MultiboxChipRemove,
  MultiboxChips,
  MultiboxContent,
  MultiboxEmpty,
  MultiboxInput,
  MultiboxItem,
  MultiboxList,
  MultiboxTrigger,
  useMultiboxAnchor,
} from "@/components/ui/multibox";
import { trpc } from "@/lib/trpc/client";
import { Radio } from "@base-ui/react/radio";
import { RadioGroup } from "@base-ui/react/radio-group";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { CaretRight, Check, Magnifier, Minus, Plus, Trash } from "@unkey/icons";
import { match } from "@unkey/match";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  FormInput,
  FormSelect,
  cn,
} from "@unkey/ui";
import { type ReactNode, useId, useState } from "react";
import { Controller, useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { DrainEndpointRow } from "./drain-endpoint-row";
import { type DrainFormValues, emptyHeaderRow } from "./drain-schema";
import {
  type SourceFilters,
  buildSourceTree,
  encodeSources,
  environmentIdsOf,
  tickedEnvironmentIds,
} from "./gateway-sources";

export function NameField() {
  const { register, formState } = useFormContext<DrainFormValues>();

  return (
    <FormInput
      requirement="required"
      label="Name"
      description="Shown in the log drain list and on the drain page."
      className="[&_input:first-of-type]:h-[36px]"
      placeholder="Production audit logs"
      error={formState.errors.name?.message}
      {...register("name")}
    />
  );
}

export function StreamField({ disabled = false }: { disabled?: boolean }) {
  const { control } = useFormContext<DrainFormValues>();
  return (
    <Controller
      control={control}
      name="stream"
      render={({ field }) => (
        <FormSelect
          label="Stream"
          value={field.value}
          onValueChange={field.onChange}
          disabled={disabled}
          options={[
            { value: "audit_logs", label: "Audit logs" },
            { value: "key_verifications", label: "Key verifications" },
            { value: "gateway_requests", label: "Gateway HTTP requests" },
            { value: "runtime_logs", label: "Runtime logs" },
            { value: "ratelimits", label: "Rate limits" },
          ]}
        />
      )}
    />
  );
}

export function EventTypesField() {
  const { control } = useFormContext<DrainFormValues>();
  const stream = useWatch({ control, name: "stream" });

  return match(stream)
    .with("audit_logs", () => <AuditEventTypesField />)
    .with("key_verifications", () => (
      <>
        <VerificationKeyspacesField />
        <VerificationOutcomesField />
      </>
    ))
    .with("gateway_requests", () => (
      <>
        <SourcesField stream="gateway_requests" />
        <GatewayStatusesField />
      </>
    ))
    .with("runtime_logs", () => <RuntimeFields />)
    .with("ratelimits", () => <RatelimitFields />)
    .exhaustive();
}

function RatelimitFields() {
  const { control } = useFormContext<DrainFormValues>();
  const namespaces = trpc.ratelimit.namespace.list.useQuery();
  const labels = new Map(namespaces.data?.map((namespace) => [namespace.id, namespace.name]));
  return (
    <>
      <p className="text-xs text-gray-9">
        Logs must match each selected filter. Leave a filter empty to send all values.
      </p>
      <Controller
        control={control}
        name="namespaceIds"
        render={({ field }) => (
          <FilterChoices
            {...field}
            options={[...labels.keys()]}
            label="Namespaces"
            searchLabel="Search namespaces"
            placeholder="All namespaces"
            emptyMessage={
              namespaces.error
                ? "Unable to load namespaces."
                : namespaces.isLoading
                  ? "Loading…"
                  : "No namespaces found."
            }
            getLabel={(id) => labels.get(id) ?? "Unknown namespace"}
          />
        )}
      />
      <Controller
        control={control}
        name="passed"
        render={({ field }) => (
          <FilterChoices
            value={field.value.map(String)}
            onChange={(values) => field.onChange(values.map((value) => value === "true"))}
            onBlur={field.onBlur}
            options={["true", "false"]}
            getLabel={(value) => (value === "true" ? "Passed" : "Blocked")}
            label="Results"
            searchLabel="Search results"
            placeholder="All results"
            emptyMessage="No results found."
          />
        )}
      />
    </>
  );
}

function RuntimeFields() {
  const { control } = useFormContext<DrainFormValues>();
  return (
    <>
      <SourcesField stream="runtime_logs" />
      <Controller
        control={control}
        name="severities"
        render={({ field }) => (
          <FilterChoices
            {...field}
            options={["error", "warn", "info", "debug"]}
            label="Severity"
            description="Choose exact severities. Leave empty to send all severities."
            searchLabel="Search severities"
            placeholder="All severities"
            emptyMessage="No severities found."
          />
        )}
      />
    </>
  );
}

function SourcesField({ stream }: { stream: "gateway_requests" | "runtime_logs" }) {
  const modeField = stream === "runtime_logs" ? "runtimeSourceMode" : "sourceMode";
  const projectField = stream === "runtime_logs" ? "runtimeProjectIds" : "projectIds";
  const appField = stream === "runtime_logs" ? "runtimeAppIds" : "appIds";
  const environmentField = stream === "runtime_logs" ? "runtimeEnvironmentIds" : "environmentIds";
  const { control, formState, setValue } = useFormContext<DrainFormValues>();
  const sourceMode = useWatch({ control, name: modeField });
  const projectIds = useWatch({ control, name: projectField });
  const appIds = useWatch({ control, name: appField });
  const environmentIds = useWatch({ control, name: environmentField });
  const [query, setQuery] = useState("");
  const [collapsed, setCollapsed] = useState<string[]>([]);
  const [replacement, setReplacement] = useState<(SourceFilters & { mode: "all" | "some" }) | null>(
    null,
  );
  const projects = trpc.deploy.project.list.useQuery();
  const environments = trpc.deploy.environment.listAll.useQuery();
  const tree = buildSourceTree(projects.data ?? [], environments.data ?? []);
  const allIds = environmentIdsOf(tree);
  const unavailableFilters = [
    ...projectIds
      .filter(
        (id) =>
          !tree.some(
            (project) =>
              project.id === id && project.apps.some((app) => app.environments.length > 0),
          ),
      )
      .map((id) => ({ type: "Project", id })),
    ...appIds
      .filter(
        (id) =>
          !tree.some((project) =>
            project.apps.some((app) => app.id === id && app.environments.length > 0),
          ),
      )
      .map((id) => ({ type: "App", id })),
    ...environmentIds
      .filter((id) => !allIds.includes(id))
      .map((id) => ({ type: "Environment", id })),
  ];
  const selected = new Set(
    tickedEnvironmentIds(tree, sourceMode, { projectIds, appIds, environmentIds }),
  );
  const error = formState.errors[environmentField]?.message;
  const unavailable =
    Boolean(projects.error || environments.error) || projects.isLoading || environments.isLoading;

  const applySelection = (selection: SourceFilters & { mode: "all" | "some" }) => {
    if (unavailable) {
      return;
    }
    const options = { shouldDirty: true, shouldValidate: true } as const;
    setValue(modeField, selection.mode, options);
    setValue(projectField, selection.projectIds, options);
    setValue(appField, selection.appIds, options);
    setValue(environmentField, selection.environmentIds, options);
  };

  const choose = (next: Set<string>) => {
    if (unavailable) {
      return;
    }
    const selection = { ...encodeSources(tree, next), mode: "some" as const };
    if (sourceMode === "some" && unavailableFilters.length > 0) {
      setReplacement(selection);
      return;
    }
    applySelection(selection);
  };

  const toggle = (ids: string[]) => {
    const next = new Set(selected);
    if (ids.every((id) => next.has(id))) {
      for (const id of ids) {
        next.delete(id);
      }
    } else {
      for (const id of ids) {
        next.add(id);
      }
    }
    choose(next);
  };

  const term = query.trim().toLowerCase();
  const matches = (text: string) => text.toLowerCase().includes(term);
  const visible = tree
    .map((project) => ({
      project,
      apps: project.apps.filter(
        (app) =>
          term === "" ||
          matches(project.name) ||
          matches(app.name) ||
          app.environments.some((environment) => matches(environment.name)),
      ),
    }))
    .filter(({ apps }) => apps.length > 0);

  const notice = sourcesNotice({
    failed: Boolean(projects.error || environments.error),
    loading: projects.isLoading || environments.isLoading,
    empty: visible.length === 0,
  });

  return (
    <fieldset disabled={unavailable} className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">Sources</legend>
      <span className="text-xs text-gray-9">
        {sourceMode === "all"
          ? "All sources in this workspace. No project, app, or environment restrictions."
          : "Select at least one project, app, or environment."}{" "}
        {stream === "runtime_logs"
          ? "Severity filters still apply."
          : "HTTP status filters still apply."}
      </span>
      <RadioGroup
        aria-label="Source scope"
        value={sourceMode}
        onValueChange={(mode) => {
          if (mode !== "all" && mode !== "some") {
            return;
          }
          const selection = { projectIds, appIds, environmentIds, mode };
          if (mode === "all" && sourceMode === "some" && unavailableFilters.length > 0) {
            setReplacement(selection);
          } else {
            applySelection(selection);
          }
        }}
        className="mt-1.5 grid gap-2 sm:grid-cols-2"
      >
        {[
          { id: "all", title: "All sources" },
          { id: "some", title: "Specific sources" },
        ].map((option) => (
          <Radio.Root
            key={option.id}
            value={option.id}
            className="group flex items-center gap-3 rounded-lg border border-grayA-4 px-3 py-2.5 transition-colors duration-150 ease-out focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent-7 data-checked:border-grayA-8 data-checked:bg-grayA-2"
          >
            <span className="flex size-4 shrink-0 items-center justify-center rounded-full border border-gray-7 transition-colors duration-150 ease-out group-data-checked:border-accent-12">
              <Radio.Indicator className="size-2 rounded-full bg-accent-12" />
            </span>
            <span className="text-[13px] text-accent-12">{option.title}</span>
          </Radio.Root>
        ))}
      </RadioGroup>
      {sourceMode === "some" ? (
        <div className="mt-1.5 overflow-hidden rounded-lg border border-gray-5">
          <div className="flex items-center gap-2 border-b border-gray-4 px-2.5 py-2">
            <Magnifier iconSize="sm-regular" className="shrink-0 text-gray-9" />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search projects, apps, environments"
              aria-label="Search sources"
              className="w-full bg-transparent text-[13px] text-accent-12 placeholder:text-gray-9 focus:outline-hidden"
            />
          </div>

          <div className="max-h-[264px] overflow-y-auto py-1">
            {notice ? <p className="px-3 py-2 text-xs text-gray-9">{notice}</p> : null}
            {visible.map(({ project, apps }) => {
              const projectEnvironmentIds = project.apps.flatMap((app) =>
                app.environments.map((environment) => environment.id),
              );
              const expanded = !collapsed.includes(project.id) || term !== "";
              return (
                <div key={project.id}>
                  <SourceRow
                    depth={0}
                    checked={checkedState(projectEnvironmentIds, selected)}
                    label={project.name}
                    meta={countLabel(project.apps.length, "app")}
                    expanded={expanded}
                    onExpand={() =>
                      setCollapsed(
                        collapsed.includes(project.id)
                          ? collapsed.filter((id) => id !== project.id)
                          : [...collapsed, project.id],
                      )
                    }
                    onToggle={() => toggle(projectEnvironmentIds)}
                  />
                  {expanded
                    ? apps.map((app) => (
                        <div key={app.id}>
                          <SourceRow
                            depth={1}
                            checked={checkedState(
                              app.environments.map((environment) => environment.id),
                              selected,
                            )}
                            label={app.name}
                            meta={countLabel(app.environments.length, "environment")}
                            onToggle={() =>
                              toggle(app.environments.map((environment) => environment.id))
                            }
                          />
                          {app.environments.map((environment) => (
                            <SourceRow
                              key={environment.id}
                              depth={2}
                              checked={selected.has(environment.id) ? "on" : "off"}
                              label={environment.name}
                              onToggle={() => toggle([environment.id])}
                            />
                          ))}
                        </div>
                      ))
                    : null}
                </div>
              );
            })}
          </div>

          <div className="flex items-center justify-between border-t border-gray-4 bg-grayA-2 px-3 py-2">
            <span className="text-xs text-gray-11">
              {`${selected.size} of ${countLabel(allIds.length, "environment")}`}
            </span>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => choose(new Set(allIds))}
                className="text-xs text-gray-11 underline underline-offset-2 hover:text-accent-12"
              >
                Select all
              </button>
              <button
                type="button"
                onClick={() => choose(new Set())}
                className="text-xs text-gray-11 underline underline-offset-2 hover:text-accent-12"
              >
                Clear all
              </button>
            </div>
          </div>
        </div>
      ) : null}
      {error ? (
        <span role="alert" className="text-xs text-error-11">
          {error}
        </span>
      ) : null}
      <AlertDialog
        open={replacement !== null}
        onOpenChange={(open) => {
          if (!open) {
            setReplacement(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Replace source filters?</AlertDialogTitle>
            <AlertDialogDescription>
              Some saved resources are unavailable in this tree. This change replaces your current
              source filters, including the unavailable filters below. Future deliveries may include
              different resources or skip retained logs. Changes take effect when you save.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <ul className="max-h-40 overflow-y-auto text-xs text-gray-11">
            {unavailableFilters.map(({ type, id }) => (
              <li key={`${type}:${id}`}>
                {type}: <code className="break-all">{id}</code>
              </li>
            ))}
          </ul>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep current filters</AlertDialogCancel>
            <AlertDialogAction
              disabled={unavailable}
              onClick={() => {
                if (replacement) {
                  applySelection(replacement);
                }
              }}
            >
              Replace filters
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </fieldset>
  );
}

function sourcesNotice({
  failed,
  loading,
  empty,
}: {
  failed: boolean;
  loading: boolean;
  empty: boolean;
}): string | null {
  if (failed) {
    return "Unable to load sources.";
  }
  if (loading) {
    return "Loading sources…";
  }
  if (empty) {
    return "No matches found.";
  }
  return null;
}

function countLabel(count: number, noun: string) {
  return `${count} ${noun}${count === 1 ? "" : "s"}`;
}

type CheckedState = "on" | "off" | "some";

function checkedState(ids: string[], selected: Set<string>): CheckedState {
  const on = ids.filter((id) => selected.has(id)).length;
  if (on === 0) {
    return "off";
  }
  return on === ids.length ? "on" : "some";
}

function SourceRow({
  depth,
  checked,
  label,
  meta,
  expanded,
  onExpand,
  onToggle,
}: {
  depth: number;
  checked: CheckedState;
  label: string;
  meta?: string;
  expanded?: boolean;
  onExpand?: () => void;
  onToggle: () => void;
}) {
  return (
    <div
      className="flex items-center gap-2 py-1 pr-3 hover:bg-grayA-2"
      style={{ paddingLeft: 10 + depth * 18 }}
    >
      {onExpand ? (
        <button
          type="button"
          aria-label={expanded ? `Collapse ${label}` : `Expand ${label}`}
          onClick={onExpand}
          className="flex size-4 shrink-0 items-center justify-center text-gray-9 hover:text-accent-12"
        >
          <CaretRight
            iconSize="sm-regular"
            className={cn("transition-transform duration-150 ease-out", expanded && "rotate-90")}
          />
        </button>
      ) : (
        <span className="size-4 shrink-0" />
      )}

      <button
        type="button"
        // biome-ignore lint/a11y/useSemanticElements: a native checkbox cannot carry the indeterminate styling this row needs
        role="checkbox"
        aria-checked={checked === "some" ? "mixed" : checked === "on"}
        onClick={onToggle}
        className="flex min-w-0 flex-1 items-center gap-2 py-0.5 text-left"
      >
        <span
          className={cn(
            "flex size-4 shrink-0 items-center justify-center rounded border transition-colors duration-150 ease-out",
            checked === "off"
              ? "border-gray-7"
              : "border-accent-12 bg-accent-12 text-white dark:text-black",
          )}
        >
          {checked === "on" ? <Check iconSize="sm-regular" /> : null}
          {checked === "some" ? <Minus iconSize="sm-regular" /> : null}
        </span>
        <span
          className={cn("truncate text-[13px]", depth === 0 ? "text-accent-12" : "text-gray-11")}
        >
          {label}
        </span>
        {meta ? <span className="ml-auto shrink-0 text-[11px] text-gray-9">{meta}</span> : null}
      </button>
    </div>
  );
}

const STATUS_MODES: { id: DrainFormValues["statusMode"]; title: string }[] = [
  { id: "all", title: "All statuses" },
  { id: "custom", title: "Custom" },
];

function GatewayStatusesField() {
  const { control, formState, setValue } = useFormContext<DrainFormValues>();
  const mode = useWatch({ control, name: "statusMode" });
  const error = formState.errors.statusClasses?.message;

  const chooseMode = (next: DrainFormValues["statusMode"]) => {
    const options = { shouldDirty: true, shouldValidate: true } as const;
    setValue("statusMode", next, options);
    if (next !== "custom") {
      setValue("statusClasses", [], options);
    }
  };

  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">HTTP statuses</legend>

      <div role="radiogroup" aria-label="Status scope" className="mt-1.5 grid grid-cols-2 gap-2">
        {STATUS_MODES.map((option) => (
          <ModeCard
            key={option.id}
            active={option.id === mode}
            title={option.title}
            onSelect={() => chooseMode(option.id)}
          />
        ))}
      </div>

      {mode === "custom" ? (
        <div className="mt-2 flex flex-col gap-1.5 duration-200 ease-out animate-in fade-in motion-reduce:animate-none">
          <Controller
            control={control}
            name="statusClasses"
            render={({ field }) => (
              <ChoiceMultibox
                value={field.value.map(String)}
                onChange={(values) => field.onChange(values.map(Number))}
                onBlur={field.onBlur}
                options={["2", "3", "4", "5"]}
                getLabel={(value) => `${value}xx`}
                searchLabel="Search HTTP statuses"
                placeholder="Choose status classes"
                emptyMessage="No HTTP statuses found."
              />
            )}
          />
          {error ? (
            <span role="alert" className="text-xs text-error-11">
              {error}
            </span>
          ) : null}
        </div>
      ) : null}
    </fieldset>
  );
}

function ModeCard({
  active,
  title,
  onSelect,
}: {
  active: boolean;
  title: string;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      // biome-ignore lint/a11y/useSemanticElements: a native radio cannot carry the card styling this control needs
      role="radio"
      aria-checked={active}
      onClick={onSelect}
      className={cn(
        "flex items-center gap-2.5 rounded-lg border px-2.5 py-2 text-left transition-colors duration-150 ease-out focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent-7",
        active ? "border-grayA-8 bg-grayA-2" : "border-grayA-4",
      )}
    >
      <span
        className={cn(
          "flex size-4 shrink-0 items-center justify-center rounded-full border transition-colors duration-150 ease-out",
          active ? "border-accent-12" : "border-gray-7",
        )}
      >
        {active ? <span className="size-2 rounded-full bg-accent-12" /> : null}
      </span>
      <span className="truncate text-[13px] text-accent-12">{title}</span>
    </button>
  );
}
const eventTypeModes = [
  { id: "all", title: "All event types" },
  { id: "specific", title: "Specific event types" },
] satisfies { id: DrainFormValues["eventTypesMode"]; title: string }[];

function AuditEventTypesField() {
  const { control, formState, setValue } = useFormContext<DrainFormValues>();
  const mode = useWatch({ control, name: "eventTypesMode" });
  const eventTypes = useWatch({ control, name: "eventTypes" });
  const error = formState.errors.eventTypes?.message;
  const statusId = useId();
  const [query, setQuery] = useState("");
  const [expandedCategories, setExpandedCategories] = useState<string[]>([]);
  const categories = new Map<string, string[]>();
  for (const eventType of new Set([...unkeyAuditLogEvents.options, ...eventTypes])) {
    const separator = eventType.indexOf(".");
    const category = separator === -1 ? eventType : eventType.slice(0, separator);
    const actions = categories.get(category) ?? [];
    actions.push(eventType);
    categories.set(category, actions);
  }
  const selected = new Set(eventTypes);
  const term = query.trim().toLowerCase();
  const visibleCategories = [...categories]
    .map(([category, actions]) => ({
      category,
      actions,
      visibleActions: actions.filter((action) => action.toLowerCase().includes(term)),
    }))
    .filter(({ visibleActions }) => visibleActions.length > 0);
  const toggle = (actions: string[]) => {
    const next = new Set(selected);
    if (actions.every((action) => next.has(action))) {
      for (const action of actions) {
        next.delete(action);
      }
    } else {
      for (const action of actions) {
        next.add(action);
      }
    }
    setValue("eventTypes", [...next], { shouldDirty: true, shouldValidate: true });
  };
  const sendingSummary =
    eventTypes.length > 0
      ? `Sending ${eventTypes.length} of ${unkeyAuditLogEvents.options.length} event types.`
      : null;
  const status = error ?? sendingSummary;

  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">Event types</legend>
      <span className="text-xs text-gray-9">
        Choose which audit events to send.{" "}
        <a
          href="https://www.unkey.com/docs/audit-log/types"
          target="_blank"
          rel="noopener noreferrer"
          className="underline underline-offset-2"
        >
          View event types
        </a>
      </span>
      <RadioGroup
        aria-label="Event type scope"
        value={mode}
        onValueChange={(next) =>
          setValue("eventTypesMode", next, { shouldValidate: true, shouldDirty: true })
        }
        className="mt-1.5 grid gap-2 sm:grid-cols-2"
      >
        {eventTypeModes.map((option) => (
          <Radio.Root
            key={option.id}
            value={option.id}
            className="group flex items-center gap-3 rounded-lg border border-grayA-4 px-3 py-2.5 transition-colors duration-150 ease-out focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent-7 data-checked:border-grayA-8 data-checked:bg-grayA-2"
          >
            <span className="flex size-4 shrink-0 items-center justify-center rounded-full border border-gray-7 transition-colors duration-150 ease-out group-data-checked:border-accent-12">
              <Radio.Indicator className="size-2 rounded-full bg-accent-12" />
            </span>
            <span className="text-[13px] text-accent-12">{option.title}</span>
          </Radio.Root>
        ))}
      </RadioGroup>

      {mode === "specific" ? (
        <div className="mt-1.5 flex flex-col gap-1.5 duration-200 ease-out animate-in fade-in motion-reduce:animate-none">
          <div className="overflow-hidden rounded-lg border border-gray-5">
            <div className="flex items-center gap-2 border-b border-gray-4 px-2.5 py-2">
              <Magnifier iconSize="sm-regular" className="shrink-0 text-gray-9" />
              <input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search categories and actions"
                aria-label="Search event types"
                aria-invalid={Boolean(error)}
                aria-describedby={status ? statusId : undefined}
                className="w-full bg-transparent text-[13px] text-accent-12 placeholder:text-gray-9 focus:outline-hidden"
              />
            </div>
            <div className="max-h-[264px] overflow-y-auto py-1">
              {visibleCategories.length === 0 ? (
                <p className="px-3 py-2 text-xs text-gray-9">No event types found.</p>
              ) : null}
              {visibleCategories.map(({ category, actions, visibleActions }) => {
                const expanded = expandedCategories.includes(category) || term !== "";
                return (
                  <div key={category}>
                    <SourceRow
                      depth={0}
                      checked={checkedState(actions, selected)}
                      label={category}
                      meta={countLabel(actions.length, "action")}
                      expanded={expanded}
                      onExpand={() =>
                        setExpandedCategories(
                          expandedCategories.includes(category)
                            ? expandedCategories.filter((value) => value !== category)
                            : [...expandedCategories, category],
                        )
                      }
                      onToggle={() => toggle(actions)}
                    />
                    {expanded
                      ? visibleActions.map((action) => (
                          <SourceRow
                            key={action}
                            depth={1}
                            checked={selected.has(action) ? "on" : "off"}
                            label={action}
                            onToggle={() => toggle([action])}
                          />
                        ))
                      : null}
                  </div>
                );
              })}
            </div>
          </div>
          {status ? (
            <span
              id={statusId}
              role={error ? "alert" : undefined}
              className={cn("text-xs", error ? "text-error-11" : "text-gray-9")}
            >
              {status}
            </span>
          ) : null}
        </div>
      ) : null}
    </fieldset>
  );
}

function VerificationOutcomesField() {
  const { control } = useFormContext<DrainFormValues>();
  return (
    <Controller
      control={control}
      name="outcomes"
      render={({ field }) => (
        <FilterChoices
          {...field}
          options={KEY_VERIFICATION_OUTCOMES}
          label="Outcomes"
          description="Choose which verification outcomes to send. Leave empty to send all outcomes."
          searchLabel="Search outcomes"
          placeholder="All outcomes"
          emptyMessage="No outcomes found."
        />
      )}
    />
  );
}

function VerificationKeyspacesField() {
  const { control } = useFormContext<DrainFormValues>();
  const {
    data: keyspaces = {},
    isLoading,
    error,
  } = trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();
  return (
    <Controller
      control={control}
      name="keySpaceIds"
      render={({ field }) => (
        <FilterChoices
          {...field}
          options={Object.keys(keyspaces)}
          label="Keyspaces"
          description="Choose which keyspaces to send verifications from. Leave empty to send all keyspaces."
          searchLabel="Search keyspaces"
          placeholder="All keyspaces"
          emptyMessage={
            error
              ? "Unable to load keyspaces."
              : isLoading
                ? "Loading keyspaces…"
                : "No keyspaces found."
          }
          getLabel={(id) => (keyspaces[id] ? `${keyspaces[id].api.name} (${id})` : id)}
        />
      )}
    />
  );
}

type ChoiceMultiboxProps = {
  value: string[];
  onChange: (value: string[]) => void;
  onBlur: () => void;
  options: readonly string[];
  searchLabel: string;
  placeholder: string;
  emptyMessage: string;
  getLabel?: (choice: string) => string;
  className?: string;
  invalid?: boolean;
  describedBy?: string;
};

function ChoiceMultibox({
  value,
  onChange,
  onBlur,
  options,
  searchLabel,
  placeholder,
  emptyMessage,
  getLabel = (choice) => choice || "Unspecified",
  className,
  invalid,
  describedBy,
}: ChoiceMultiboxProps) {
  const anchor = useMultiboxAnchor();
  const choices = Array.from(new Set([...options, ...value]));

  return (
    <Multibox items={choices} value={value} onValueChange={onChange} itemToStringLabel={getLabel}>
      <MultiboxChips ref={anchor} className={className}>
        {value.map((choice) => (
          <MultiboxChip key={choice}>
            <span className="font-mono">{getLabel(choice)}</span>
            <MultiboxChipRemove />
          </MultiboxChip>
        ))}
        <MultiboxInput
          aria-label={searchLabel}
          aria-invalid={invalid}
          aria-describedby={describedBy}
          placeholder={value.length === 0 ? placeholder : "Search"}
          onBlur={onBlur}
        />
        <MultiboxTrigger />
      </MultiboxChips>
      <MultiboxContent anchor={anchor}>
        <MultiboxEmpty>{emptyMessage}</MultiboxEmpty>
        <MultiboxList>
          {(choice: string) => (
            <MultiboxItem key={choice} value={choice}>
              <span className="font-mono text-xs">{getLabel(choice)}</span>
            </MultiboxItem>
          )}
        </MultiboxList>
      </MultiboxContent>
    </Multibox>
  );
}

function FilterChoices({
  label,
  description,
  ...choices
}: ChoiceMultiboxProps & {
  label: string;
  description?: ReactNode;
}) {
  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">{label}</legend>
      {description ? <span className="text-xs text-gray-9">{description}</span> : null}
      <ChoiceMultibox {...choices} className="mt-1.5" />
    </fieldset>
  );
}

export function HeaderFields() {
  const { control, register, formState } = useFormContext<DrainFormValues>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: "headers",
  });
  const errors = formState.errors.headers;

  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">Headers</legend>
      <span className="text-xs text-gray-9">
        Optional. Unkey encrypts header values before storing them, and hides them afterwards.
      </span>
      <div className="mt-1.5 flex flex-col gap-3">
        {fields.map((field, index) => (
          <div key={field.id} className="flex items-start gap-3">
            <FormInput
              label="Name"
              placeholder="Authorization"
              className="flex-1 [&_input:first-of-type]:h-[36px]"
              // A stored header is addressed by name on save, so renaming it cannot mean anything.
              readOnly={field.stored}
              error={errors?.[index]?.name?.message}
              {...register(`headers.${index}.name`)}
            />
            <FormInput
              label="Value"
              type="password"
              autoComplete="off"
              placeholder={field.stored ? "•••••••••• unchanged" : "Bearer …"}
              className="flex-1 [&_input:first-of-type]:h-[36px]"
              error={errors?.[index]?.value?.message}
              {...register(`headers.${index}.value`)}
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="mt-[26px] size-9 shrink-0 justify-center px-0 text-gray-11"
              aria-label={`Remove header ${index + 1}`}
              onClick={() => remove(index)}
            >
              <Trash iconSize="sm-regular" />
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          className="w-fit"
          disabled={fields.length >= 32}
          onClick={() => append({ ...emptyHeaderRow })}
        >
          <Plus iconSize="sm-regular" />
          Add header
        </Button>
      </div>
    </fieldset>
  );
}

function HttpFields() {
  return (
    <>
      <DrainEndpointRow />
      <HeaderFields />
    </>
  );
}

function AxiomFields({ tokenRequired }: { tokenRequired: boolean }) {
  const { register, formState } = useFormContext<DrainFormValues>();

  return (
    <>
      <FormInput
        requirement="required"
        label="Dataset"
        description="The Axiom dataset that receives this stream."
        className="[&_input:first-of-type]:h-[36px]"
        placeholder="audit-logs"
        error={formState.errors.dataset?.message}
        {...register("dataset")}
      />
      <FormInput
        requirement={tokenRequired ? "required" : "optional"}
        label="Token"
        description={
          tokenRequired
            ? "Use an Axiom API token that can ingest data into this dataset."
            : "Leave blank to keep the current token."
        }
        type="password"
        autoComplete="off"
        placeholder={tokenRequired ? undefined : "•••••••••• unchanged"}
        className="[&_input:first-of-type]:h-[36px]"
        error={formState.errors.token?.message}
        {...register("token")}
      />
    </>
  );
}

export function DestinationFields({ tokenRequired }: { tokenRequired: boolean }) {
  const kind = useWatch<DrainFormValues, "kind">({ name: "kind" });

  return kind === "http" ? <HttpFields /> : <AxiomFields tokenRequired={tokenRequired} />;
}
