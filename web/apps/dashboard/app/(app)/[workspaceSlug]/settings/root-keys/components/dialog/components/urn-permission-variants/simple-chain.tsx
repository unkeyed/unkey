"use client";

import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import { Button } from "@unkey/ui";
import { useMemo, useState } from "react";
import { actionGroups, buildActions, permission, togglePermission } from "./catalog";
import type {
  ActionDefinition,
  ApiItem,
  PermissionResourceSuggestions,
  ScopedItem,
  UrnPermissionVariantProps,
} from "./types";

type SegmentMode = "wildcard" | "specific" | "terminator";

type SegmentState = {
  mode: SegmentMode;
  value: string;
};

type ResourceLevel = {
  collection: string;
  label: string;
  placeholder: string;
};

type SuggestionItem = ScopedItem & {
  projectId?: string;
  appId?: string;
  environmentId?: string;
  namespaceId?: string;
};

function defaultSegment(): SegmentState {
  return { mode: "terminator", value: "" };
}

function levelsForAction(action: ActionDefinition): ResourceLevel[] {
  if (action.id === "*") {
    return [];
  }

  if (action.group === "keys") {
    return [
      { collection: "keyspaces", label: "Keyspace", placeholder: "keyspace id" },
      { collection: "keys", label: "Key", placeholder: "key id" },
    ];
  }

  if (action.group === "deployments") {
    return [
      { collection: "projects", label: "Project", placeholder: "project id" },
      { collection: "apps", label: "App", placeholder: "app id" },
      { collection: "environments", label: "Environment", placeholder: "environment id" },
      { collection: "deployments", label: "Deployment", placeholder: "deployment id" },
    ];
  }

  if (action.group === "ratelimits" && action.terminalLabel === "Overrides") {
    return [
      { collection: "ratelimits/namespaces", label: "Namespace", placeholder: "namespace id" },
      { collection: "overrides", label: "Override", placeholder: "override id" },
    ];
  }

  if (action.group === "ratelimits") {
    return [
      { collection: "ratelimits/namespaces", label: "Namespace", placeholder: "namespace id" },
    ];
  }

  if (action.group === "authorization" && action.category === "rbac/roles") {
    return [{ collection: "rbac/roles", label: "Role", placeholder: "role id" }];
  }

  if (action.group === "authorization" && action.category === "rbac/permissions") {
    return [{ collection: "rbac/permissions", label: "Permission", placeholder: "permission id" }];
  }

  if (action.group === "identities") {
    return [{ collection: "identities", label: "Identity", placeholder: "identity id" }];
  }

  return [{ collection: "**", label: "Workspace", placeholder: "*" }];
}

function normalize(state: SegmentState): string {
  if (state.mode === "wildcard") {
    return "*";
  }
  if (state.mode === "terminator") {
    return "**";
  }
  return state.value.trim() || "*";
}

function buildResource(levels: ResourceLevel[], state: Record<number, SegmentState>): string {
  const parts: string[] = [];

  for (let index = 0; index < levels.length; index++) {
    const level = levels[index];
    const current = state[index] ?? defaultSegment();
    parts.push(level.collection);
    parts.push(normalize(current));
    if (current.mode === "terminator") {
      break;
    }
  }

  return parts.join("/");
}

function buildActionResource(
  action: ActionDefinition,
  levels: ResourceLevel[],
  state: Record<number, SegmentState>,
): string {
  if (action.id === "*") {
    return "**";
  }
  return buildResource(levels, state);
}

function buildVisibleParts(
  levels: ResourceLevel[],
  state: Record<number, SegmentState>,
): Array<{ kind: "collection" | "id"; value: string; level?: ResourceLevel; index?: number }> {
  const parts: Array<{
    kind: "collection" | "id";
    value: string;
    level?: ResourceLevel;
    index?: number;
  }> = [];

  for (let index = 0; index < levels.length; index++) {
    const level = levels[index];
    const current = state[index] ?? defaultSegment();
    parts.push({ kind: "collection", value: level.collection });
    parts.push({ kind: "id", value: normalize(current), level, index });
    if (current.mode === "terminator") {
      break;
    }
  }

  return parts;
}

function readableSegment({
  level,
  state,
  chainState,
  apis,
  projects,
  permissionResources,
}: {
  level: ResourceLevel;
  state: SegmentState;
  chainState: Record<number, SegmentState>;
  apis: UrnPermissionVariantProps["apis"];
  projects: UrnPermissionVariantProps["projects"];
  permissionResources?: PermissionResourceSuggestions;
}): string {
  const label = level.label.toLowerCase();
  if (state.mode === "wildcard") {
    return `any ${label}`;
  }
  const value = normalize(state);
  const resource = resourceItemsForLevel({
    level,
    chainState,
    apis,
    projects,
    permissionResources,
  }).find((item) => item.id === value);
  return `${label} ${resource?.name ?? value}`;
}

function readableCollection(level: ResourceLevel): string {
  return level.collection.split("/").at(-1) ?? level.collection;
}

function joinScope(segments: string[]): string {
  const [leaf, ...parents] = [...segments].reverse();
  if (!leaf) {
    return "this workspace";
  }
  return [leaf, ...parents.map((parent) => `in ${parent}`)].join(" ");
}

function buildSummary(
  action: ActionDefinition,
  levels: ResourceLevel[],
  state: Record<number, SegmentState>,
  apis: UrnPermissionVariantProps["apis"],
  projects: UrnPermissionVariantProps["projects"],
  permissionResources?: PermissionResourceSuggestions,
): { action: string; scope: string } {
  const segments: string[] = [];

  for (let index = 0; index < levels.length; index++) {
    const level = levels[index];
    const current = state[index] ?? defaultSegment();

    if (current.mode === "terminator") {
      const collection = readableCollection(level);
      const parentScope = segments.length > 0 ? ` in ${joinScope(segments)}` : "";
      return {
        action: action.id,
        scope: `for everything under ${collection}${parentScope}.`,
      };
    }

    segments.push(
      readableSegment({
        level,
        state: current,
        chainState: state,
        apis,
        projects,
        permissionResources,
      }),
    );
  }

  return {
    action: action.id,
    scope: `for ${joinScope(segments)}.`,
  };
}

export function SimpleChainVariant({
  workspaceId,
  apis,
  projects,
  permissionResources,
  selectedPermissions,
  onChange,
}: UrnPermissionVariantProps) {
  const actions = useMemo(() => buildActions(), []);
  const [actionId, setActionId] = useState(actions.at(0)?.id ?? "");
  const [stateByAction, setStateByAction] = useState<Record<string, Record<number, SegmentState>>>(
    {},
  );

  const selectedAction = actions.find((action) => action.id === actionId) ?? actions.at(0) ?? null;
  const levels = selectedAction ? levelsForAction(selectedAction) : [];
  const chainState = selectedAction ? (stateByAction[selectedAction.id] ?? {}) : {};
  const visibleParts = buildVisibleParts(levels, chainState);
  const resource = selectedAction ? buildActionResource(selectedAction, levels, chainState) : "";
  const summary = selectedAction
    ? buildSummary(selectedAction, levels, chainState, apis, projects, permissionResources)
    : null;
  const nextPermission = selectedAction
    ? permission(workspaceId, resource, selectedAction.id)
    : null;
  const isSelected = nextPermission ? selectedPermissions.includes(nextPermission) : false;

  function setSegment(index: number, next: SegmentState) {
    if (!selectedAction) {
      return;
    }
    setStateByAction((previous) => {
      const nextState: Record<number, SegmentState> = {};
      for (const [key, value] of Object.entries(previous[selectedAction.id] ?? {})) {
        const segmentIndex = Number(key);
        if (segmentIndex < index) {
          nextState[segmentIndex] = value;
        }
      }
      nextState[index] = next;

      return {
        ...previous,
        [selectedAction.id]: nextState,
      };
    });
  }

  return (
    <div className="flex min-h-[420px] flex-col gap-5 rounded-lg border border-grayA-4 bg-white p-5 dark:bg-black">
      <section className="grid gap-3 lg:grid-cols-[320px_minmax(0,1fr)] lg:items-start">
        <div>
          <div className="text-sm font-medium text-gray-12">Action</div>
          <select
            value={actionId}
            className="mt-3 h-10 w-full rounded-md border border-grayA-5 bg-white px-3 text-sm text-gray-12 dark:bg-black"
            onChange={(event) => setActionId(event.target.value)}
          >
            {actionGroups.map((group) => (
              <optgroup key={group.id} label={group.label}>
                {actions
                  .filter((action) => action.group === group.id)
                  .map((action) => (
                    <option key={action.id} value={action.id}>
                      {action.label}
                    </option>
                  ))}
              </optgroup>
            ))}
          </select>
        </div>
        <div>
          <div className="text-sm font-medium text-gray-12">Description</div>
          {selectedAction ? (
            <div className="mt-3 rounded-md bg-grayA-2 p-3 text-sm text-gray-10">
              {selectedAction.description}
            </div>
          ) : null}
        </div>
      </section>

      <section className="min-w-0">
        <div className="text-sm font-medium text-gray-12">Resource path</div>
        <div className="mt-3 overflow-x-auto rounded-lg border border-grayA-4 bg-grayA-1 p-3">
          <div className="flex min-w-max flex-wrap items-center gap-1 font-mono text-sm">
            {visibleParts.map((part, partIndex) => (
              <div
                key={`${part.kind}-${part.value}-${partIndex}`}
                className="flex items-center gap-1"
              >
                {partIndex > 0 ? <span className="text-grayA-8">/</span> : null}
                {part.kind === "collection" ? (
                  <span className="px-1 py-2 text-gray-10">{part.value}</span>
                ) : part.level && typeof part.index === "number" ? (
                  <SegmentControl
                    level={part.level}
                    state={chainState[part.index] ?? defaultSegment()}
                    chainState={chainState}
                    canSelectSpecific={hasSpecificParents(chainState, part.index)}
                    apis={apis}
                    projects={projects}
                    permissionResources={permissionResources}
                    onChange={(next) => setSegment(part.index ?? 0, next)}
                  />
                ) : null}
              </div>
            ))}
            <span className="text-grayA-8">#</span>
            <span className="px-1 py-2 text-accent-11">{selectedAction?.id ?? "action"}</span>
          </div>
          {summary ? (
            <div className="mt-3 border-t border-grayA-3 pt-3 text-sm text-gray-11">
              Allows <span className="font-mono text-gray-12">{summary.action}</span>{" "}
              {summary.scope}
            </div>
          ) : null}
        </div>

        <div className="mt-5 rounded-lg border border-grayA-4 bg-grayA-2 p-3">
          <div className="text-[11px] font-medium uppercase text-gray-9">Generated permission</div>
          <div className="mt-2 break-all font-mono text-xs text-gray-12">{nextPermission}</div>
        </div>

        {nextPermission ? (
          <Button
            type="button"
            variant={isSelected ? "outline" : "primary"}
            className="mt-3 rounded-md"
            onClick={() => onChange(togglePermission(selectedPermissions, nextPermission))}
          >
            {isSelected ? "Remove permission" : "Add permission"}
          </Button>
        ) : null}
      </section>
    </div>
  );
}

function SegmentControl({
  level,
  state,
  chainState,
  canSelectSpecific,
  apis,
  projects,
  permissionResources,
  onChange,
}: {
  level: ResourceLevel;
  state: SegmentState;
  chainState: Record<number, SegmentState>;
  canSelectSpecific: boolean;
  apis: UrnPermissionVariantProps["apis"];
  projects: UrnPermissionVariantProps["projects"];
  permissionResources?: PermissionResourceSuggestions;
  onChange: (state: SegmentState) => void;
}) {
  const value = state.mode === "specific" ? state.value : normalize(state);
  const options = useMemo(
    () => [
      {
        value: "*",
        label: <WildcardOption label={`Any ${level.label.toLowerCase()}`} symbol="*" />,
        selectedLabel: <span className="font-sans text-gray-12">Any</span>,
        searchValue: `any ${level.label.toLowerCase()} wildcard *`,
      },
      ...(canSelectSpecific
        ? [
            {
              value: "**",
              label: <WildcardOption label="Everything" symbol="**" />,
              selectedLabel: <span className="font-sans text-gray-12">Everything</span>,
              searchValue: "everything below **",
            },
            ...suggestionsForLevel({
              level,
              chainState,
              apis,
              projects,
              permissionResources,
            }),
          ]
        : []),
    ],
    [level, chainState, canSelectSpecific, apis, projects, permissionResources],
  );

  return (
    <Combobox
      value={value}
      creatable={canSelectSpecific}
      options={options}
      onSelect={(next) => {
        if (next === "*") {
          onChange({ mode: "wildcard", value: "" });
          return;
        }
        if (next === "**") {
          onChange({ mode: "terminator", value: "" });
          return;
        }
        onChange({ mode: "specific", value: next });
      }}
      placeholder={level.placeholder}
      searchPlaceholder={
        canSelectSpecific ? `Search ${level.placeholder}` : "Select a specific parent first"
      }
      emptyMessage={
        canSelectSpecific ? "Type an id and press Enter." : "Select a specific parent first."
      }
      wrapperClassName="w-auto"
      className="h-8 min-w-[88px] max-w-[260px] rounded-md border-transparent bg-transparent px-2 pr-7 font-mono text-sm shadow-none hover:border-grayA-5 hover:bg-white focus:border-accent-8 dark:bg-transparent dark:hover:bg-black"
      popoverClassName="w-[320px]"
    />
  );
}

function WildcardOption({ label, symbol }: { label: string; symbol: "*" | "**" }) {
  return (
    <span className="flex min-w-0 items-center justify-between gap-3">
      <span className="truncate">{label}</span>
      <span className="font-mono text-gray-9">{symbol}</span>
    </span>
  );
}

function resourceOption(item: SuggestionItem): ComboboxOption {
  return {
    value: item.id,
    searchValue: `${item.id} ${item.name}`,
    selectedLabel: <span className="font-mono">{item.name}</span>,
    label: (
      <span className="flex min-w-0 flex-col">
        <span className="truncate text-gray-12">{item.name}</span>
        <span className="truncate font-mono text-gray-9">{item.id}</span>
      </span>
    ),
  };
}

function keyspaceItems(apis: UrnPermissionVariantProps["apis"]): SuggestionItem[] {
  const seen = new Set<string>();
  return apis
    .filter((api): api is ApiItem & { keyspaceId: string } => Boolean(api.keyspaceId))
    .filter((api) => {
      if (seen.has(api.keyspaceId)) {
        return false;
      }
      seen.add(api.keyspaceId);
      return true;
    })
    .map((api) => ({ id: api.keyspaceId, name: api.name }));
}

function specificSegment(state: Record<number, SegmentState>, index: number): string | null {
  const segment = state[index];
  if (segment?.mode !== "specific") {
    return null;
  }
  return segment.value.trim() || null;
}

function hasSpecificParents(state: Record<number, SegmentState>, index: number): boolean {
  for (let parentIndex = 0; parentIndex < index; parentIndex++) {
    if (!specificSegment(state, parentIndex)) {
      return false;
    }
  }
  return true;
}

function suggestionsForLevel({
  level,
  chainState,
  apis,
  projects,
  permissionResources,
}: {
  level: ResourceLevel;
  chainState: Record<number, SegmentState>;
  apis: UrnPermissionVariantProps["apis"];
  projects: UrnPermissionVariantProps["projects"];
  permissionResources?: PermissionResourceSuggestions;
}): ComboboxOption[] {
  return resourceItemsForLevel({
    level,
    chainState,
    apis,
    projects,
    permissionResources,
  }).map(resourceOption);
}

function resourceItemsForLevel({
  level,
  chainState,
  apis,
  projects,
  permissionResources,
}: {
  level: ResourceLevel;
  chainState: Record<number, SegmentState>;
  apis: UrnPermissionVariantProps["apis"];
  projects: UrnPermissionVariantProps["projects"];
  permissionResources?: PermissionResourceSuggestions;
}): SuggestionItem[] {
  const ancestor0Id = specificSegment(chainState, 0);
  const ancestor1Id = specificSegment(chainState, 1);
  const ancestor2Id = specificSegment(chainState, 2);

  if (level.collection === "keyspaces") {
    return keyspaceItems(apis);
  }
  if (level.collection === "projects") {
    return permissionResources?.projects ?? projects;
  }
  if (level.collection === "apps") {
    return (permissionResources?.apps ?? []).filter(
      (app) => !ancestor0Id || app.projectId === ancestor0Id,
    );
  }
  if (level.collection === "environments") {
    return (permissionResources?.environments ?? [])
      .filter((environment) => !ancestor0Id || environment.projectId === ancestor0Id)
      .filter((environment) => !ancestor1Id || environment.appId === ancestor1Id);
  }
  if (level.collection === "deployments") {
    return (permissionResources?.deployments ?? [])
      .filter((deployment) => !ancestor0Id || deployment.projectId === ancestor0Id)
      .filter((deployment) => !ancestor1Id || deployment.appId === ancestor1Id)
      .filter((deployment) => !ancestor2Id || deployment.environmentId === ancestor2Id);
  }
  if (level.collection === "ratelimits/namespaces") {
    return permissionResources?.ratelimitNamespaces ?? [];
  }
  if (level.collection === "overrides") {
    return (permissionResources?.ratelimitOverrides ?? []).filter(
      (override) => !ancestor0Id || override.namespaceId === ancestor0Id,
    );
  }
  if (level.collection === "rbac/roles") {
    return permissionResources?.roles ?? [];
  }
  if (level.collection === "rbac/permissions") {
    return permissionResources?.permissions ?? [];
  }
  if (level.collection === "identities") {
    return permissionResources?.identities ?? [];
  }
  return [];
}
