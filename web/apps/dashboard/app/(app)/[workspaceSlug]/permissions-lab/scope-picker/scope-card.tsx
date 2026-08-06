"use client";

/**
 * Right panel: three plain-language scope choices for the selected node, an
 * action picker, a principal select, and a live summary of the permissions
 * that a grant would add. The generated URN is always visible so the grammar
 * is learned by watching, never by typing.
 */

import {
  Button,
  Checkbox,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  cn,
  toast,
} from "@unkey/ui";
import { useEffect, useMemo, useState } from "react";
import { PermissionChip, UrnText } from "../components/urn-display";
import { type ActionDef, RESOURCE_TYPES, actionsForPath } from "../lib/catalog";
import { WORKSPACE_ID, coverage, perm } from "../lib/mock-data";
import { type GrantOp, usePermissionsLab } from "../lib/store";
import { parsePermission, permissionCovers } from "../lib/urn";
import { type ScopeID, type ScopeOption, type TreeNode, scopeOptionsFor } from "./model";

export function ScopeCard({ node }: { node: TreeNode }) {
  const lab = usePermissionsLab();
  const [scopeID, setScopeID] = useState<ScopeID>("exact");
  const [selectedActions, setSelectedActions] = useState<ReadonlySet<string>>(new Set());
  const [principalID, setPrincipalID] = useState<string | null>(null);

  // A new node starts from the narrowest scope with a clean action slate.
  // biome-ignore lint/correctness/useExhaustiveDependencies: node.path is the reset trigger, not a read dependency
  useEffect(() => {
    setScopeID("exact");
    setSelectedActions(new Set());
  }, [node.path]);

  const options = useMemo(() => scopeOptionsFor(node), [node]);
  const fallback = options.length > 0 ? options[0] : null;
  const selected = options.find((o) => o.id === scopeID && o.disabledReason === null) ?? fallback;
  const pattern = selected ? selected.pattern : "";

  // Flipping the scope can change the action catalog (a descendant scope is a
  // union of types); drop selections that are no longer valid.
  useEffect(() => {
    if (pattern === "") {
      return;
    }
    const valid = new Set(actionsForPath(pattern).map((a) => a.action));
    setSelectedActions((prev) => {
      const kept = [...prev].filter((action) => valid.has(action));
      return kept.length === prev.size ? prev : new Set(kept);
    });
  }, [pattern]);

  const resource = node.resource;
  if (!resource || !selected) {
    return null;
  }

  const typeLabel = RESOURCE_TYPES.find((t) => t.type === resource.type)?.label ?? resource.type;
  const actionDefs = actionsForPath(pattern);
  const available = actionDefs.filter((a) => a.implemented);
  const planned = actionDefs.filter((a) => !a.implemented);

  const principal = lab.state.principals.find((p) => p.id === principalID) ?? null;
  const principalRoles = principal
    ? lab.state.roles.filter((role) => principal.roles.includes(role.id))
    : [];

  const effectiveGrants = principal
    ? lab.effectivePermissions(principal.id).flatMap((value) => {
        const parsed = parsePermission(value);
        return parsed.ok ? [parsed.value] : [];
      })
    : [];

  const preview = [...selectedActions].sort().map((action) => {
    const permission = perm(pattern, action);
    const parsed = parsePermission(permission);
    const covered =
      parsed.ok && effectiveGrants.some((grant) => permissionCovers(grant, parsed.value));
    return { action, permission, covered };
  });
  const toAdd = preview.filter((row) => !row.covered);

  const toggleAction = (action: string, checked: boolean) => {
    setSelectedActions((prev) => {
      const next = new Set(prev);
      if (checked) {
        next.add(action);
      } else {
        next.delete(action);
      }
      return next;
    });
  };

  const grant = () => {
    if (!principal || toAdd.length === 0) {
      return;
    }
    const ops: GrantOp[] = toAdd.map((row) => ({
      op: "add",
      principalID: principal.id,
      permission: row.permission,
    }));
    lab.commit(`Grant ${countNoun(toAdd.length)} on ${pattern} to ${principal.name}`, ops);
    toast.success(`Granted ${countNoun(toAdd.length)} to ${principal.name}`);
    setSelectedActions(new Set());
  };

  const removeGrant = (permission: string) => {
    if (!principal) {
      return;
    }
    lab.commit(`Remove permission from ${principal.name}`, [
      { op: "remove", principalID: principal.id, permission },
    ]);
    toast.success(`Removed permission from ${principal.name}`);
  };

  let footer: string;
  if (selectedActions.size === 0) {
    footer = "Select at least one action.";
  } else if (!principal) {
    footer = "Choose a root key to grant to.";
  } else if (toAdd.length === 0) {
    footer = `${principal.name} already has everything selected.`;
  } else {
    footer = `Will add ${countNoun(toAdd.length)} to ${principal.name}.`;
  }

  return (
    <div className="flex flex-col gap-4">
      <section className="rounded-lg border border-grayA-4">
        <header className="flex items-center justify-between gap-3 border-b border-grayA-4 px-4 py-3">
          <div className="flex min-w-0 flex-col">
            <span className="truncate text-sm font-medium text-gray-12">{resource.label}</span>
            <span className="truncate font-mono text-[11px] text-gray-9">{resource.path}</span>
          </div>
          <span className="shrink-0 rounded-full border border-grayA-4 bg-grayA-2 px-2 py-0.5 text-[11px] text-gray-11">
            {typeLabel}
          </span>
        </header>
        <div className="flex flex-col gap-2 p-4" role="radiogroup" aria-label="Scope">
          {options.map((option) => (
            <ScopeOptionRow
              key={option.id}
              option={option}
              checked={option.id === selected.id}
              onSelect={() => setScopeID(option.id)}
            />
          ))}
        </div>
      </section>

      <section className="flex flex-col gap-3 rounded-lg border border-grayA-4 p-4">
        <header className="flex items-baseline gap-2">
          <span className="text-sm font-medium text-gray-12">Actions</span>
          <span className="text-xs text-gray-9">what the grant allows on this scope</span>
        </header>
        {actionDefs.length === 0 ? (
          <p className="text-sm text-gray-10">No actions are defined for this scope.</p>
        ) : (
          <div className="flex flex-col gap-3">
            <ActionGroup
              title="Available today"
              defs={available}
              selected={selectedActions}
              onToggle={toggleAction}
            />
            <ActionGroup
              title="Planned"
              defs={planned}
              selected={selectedActions}
              onToggle={toggleAction}
            />
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3 rounded-lg border border-grayA-4 p-4">
        <header className="text-sm font-medium text-gray-12">Grant</header>
        <div className="flex flex-col gap-1.5">
          <span className="text-xs text-gray-11">Root key</span>
          <Select
            value={principalID}
            onValueChange={(value) => {
              if (value !== null) {
                setPrincipalID(value);
              }
            }}
            items={lab.state.principals.map((p) => ({ value: p.id, label: p.name }))}
          >
            <SelectTrigger className="w-full max-w-sm">
              <SelectValue placeholder="Choose a root key" />
            </SelectTrigger>
            <SelectContent>
              {lab.state.principals.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {preview.length > 0 && (
          <div className="flex flex-col gap-1.5 overflow-x-auto rounded-md bg-grayA-2 p-3">
            {preview.map((row) => (
              <div key={row.permission} className="flex items-center justify-between gap-3">
                <UrnText
                  value={row.permission}
                  className={row.covered ? "opacity-50" : undefined}
                />
                {row.covered ? (
                  <span className="whitespace-nowrap text-[11px] text-gray-9">already covered</span>
                ) : (
                  <span className="whitespace-nowrap text-[11px] text-success-11">new</span>
                )}
              </div>
            ))}
          </div>
        )}
        <div className="flex items-center justify-between gap-3">
          <span className="text-sm text-gray-11">{footer}</span>
          <Button variant="primary" disabled={!principal || toAdd.length === 0} onClick={grant}>
            {toAdd.length > 0 ? `Grant ${countNoun(toAdd.length)}` : "Grant"}
          </Button>
        </div>
      </section>

      {principal && (
        <section className="flex flex-col gap-3 rounded-lg border border-grayA-4 p-4">
          <header className="flex items-center justify-between gap-3">
            <span className="text-sm font-medium text-gray-12">
              Current grants for {principal.name}
            </span>
            <span className="font-mono text-[11px] text-gray-9">{principal.id}</span>
          </header>
          {principal.permissions.length === 0 ? (
            <p className="text-sm text-gray-10">
              No direct permissions yet. Grant one above and it shows up here.
            </p>
          ) : (
            <div className="flex flex-wrap gap-1.5">
              {principal.permissions.map((permission) => (
                <PermissionChip
                  key={permission}
                  value={permission}
                  onRemove={() => removeGrant(permission)}
                />
              ))}
            </div>
          )}
          {principalRoles.map((role) => (
            <div key={role.id} className="flex flex-col gap-1.5">
              <span className="text-[11px] uppercase tracking-wide text-gray-10">
                via role {role.name}
              </span>
              <div className="flex flex-wrap gap-1.5 opacity-80">
                {role.permissions.map((permission) => (
                  <PermissionChip key={permission} value={permission} />
                ))}
              </div>
            </div>
          ))}
        </section>
      )}
    </div>
  );
}

function ScopeOptionRow({
  option,
  checked,
  onSelect,
}: {
  option: ScopeOption;
  checked: boolean;
  onSelect: () => void;
}) {
  const disabled = option.disabledReason !== null;
  const covered = disabled ? 0 : coverage(option.pattern).length;

  return (
    <button
      type="button"
      // biome-ignore lint/a11y/useSemanticElements: radio-card option; a native radio input cannot contain this rich layout
      role="radio"
      aria-checked={checked}
      disabled={disabled}
      onClick={onSelect}
      className={cn(
        "flex items-start gap-3 rounded-lg border p-3 text-left transition-colors",
        checked ? "border-grayA-7 bg-grayA-2" : "border-grayA-4",
        disabled ? "cursor-not-allowed opacity-60" : "hover:border-grayA-6 hover:bg-grayA-2",
      )}
    >
      <span
        className={cn(
          "mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border transition-colors",
          checked ? "border-accent-12" : "border-grayA-6",
        )}
      >
        {checked && <span className="h-2 w-2 rounded-full bg-accent-12" />}
      </span>
      <span className="flex min-w-0 flex-1 flex-col gap-1">
        <span className="text-[13px] font-medium text-gray-12">{option.title}</span>
        {disabled ? (
          <span className="text-xs text-gray-10">{option.disabledReason}</span>
        ) : (
          <>
            <span className="text-xs text-gray-11">{option.detail}</span>
            <span className="overflow-x-auto pt-1">
              <PatternText
                pattern={option.pattern}
                changedSegment={option.changedSegment}
                pulse={checked}
              />
            </span>
            <span className="text-[11px] text-gray-9">
              Covers {covered === 1 ? "1 resource" : `${covered} resources`} today
            </span>
          </>
        )}
      </span>
    </button>
  );
}

function ActionGroup({
  title,
  defs,
  selected,
  onToggle,
}: {
  title: string;
  defs: ActionDef[];
  selected: ReadonlySet<string>;
  onToggle: (action: string, checked: boolean) => void;
}) {
  if (defs.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[11px] uppercase tracking-wide text-gray-10">{title}</span>
      <div className="flex flex-col">
        {defs.map((def) => (
          // biome-ignore lint/a11y/noLabelWithoutControl: wraps the Checkbox whose native labelable element is a descendant
          <label
            key={def.action}
            className="flex cursor-pointer items-start gap-2.5 rounded-md px-1.5 py-1.5 transition-colors hover:bg-grayA-2"
          >
            <span className="mt-0.5">
              <Checkbox
                checked={selected.has(def.action)}
                onCheckedChange={(checked) => onToggle(def.action, checked)}
                aria-label={`Grant ${def.action}`}
              />
            </span>
            <span className="flex min-w-0 flex-col">
              <span className="font-mono text-xs text-gray-12">{def.action}</span>
              <span className="text-xs text-gray-11">{def.description}</span>
            </span>
          </label>
        ))}
      </div>
    </div>
  );
}

/**
 * Resource URN without an action, colored like UrnText. The segment the
 * current scope changed remounts on selection (key flip) so its background
 * pulse replays, teaching which part of the URN the radio choice rewrote.
 */
function PatternText({
  pattern,
  changedSegment,
  pulse,
}: {
  pattern: string;
  changedSegment: number | null;
  pulse: boolean;
}) {
  const segments = pattern.split("/");
  return (
    <span className="whitespace-nowrap font-mono text-xs">
      <span className="text-gray-9">unkey:v1:</span>
      <span className="text-gray-10">{WORKSPACE_ID}</span>
      <span className="text-gray-9">:</span>
      {segments.map((segment, i) => {
        const isWildcard = segment === "*" || segment === "**";
        const highlight = pulse && changedSegment === i;
        return (
          <span key={`${i}-${segment}`}>
            {i > 0 && <span className="text-gray-8">/</span>}
            <span
              key={highlight ? "pulse" : "still"}
              style={highlight ? { animation: "permlab-scope-pulse 900ms ease-out" } : undefined}
              className={cn(
                "rounded-[2px]",
                isWildcard
                  ? "font-semibold text-warning-11"
                  : i % 2 === 0
                    ? "text-gray-11"
                    : "text-gray-12",
              )}
            >
              {segment}
            </span>
          </span>
        );
      })}
    </span>
  );
}

function countNoun(count: number): string {
  return count === 1 ? "1 permission" : `${count} permissions`;
}
