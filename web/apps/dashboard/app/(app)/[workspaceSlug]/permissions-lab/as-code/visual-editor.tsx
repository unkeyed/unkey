"use client";

/**
 * The visual half of the two-way editor: direct grants as removable chips
 * grouped by resource family, a validated add row, and role-derived grants in
 * a read-only section. Every edit calls straight into the shared store, which
 * regenerates the code pane.
 */

import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { useState } from "react";
import { PermissionChip, UrnText } from "../components/urn-display";
import { actionsForPath } from "../lib/catalog";
import { type MockPrincipal, type MockRole, perm } from "../lib/mock-data";
import { parsePermission, validateResourcePath } from "../lib/urn";

const FAMILY_ORDER = ["**", "keyspaces", "identities", "ratelimits", "rbac", "projects"];

const FAMILY_LABELS: Record<string, string> = {
  "**": "Global",
  keyspaces: "Keyspaces",
  identities: "Identities",
  ratelimits: "Ratelimits",
  rbac: "RBAC",
  projects: "Projects",
};

function familyOf(permission: string): string {
  const parsed = parsePermission(permission);
  if (!parsed.ok) {
    return "other";
  }
  return parsed.value.urn.resource.split("/")[0];
}

function familyRank(family: string): number {
  const rank = FAMILY_ORDER.indexOf(family);
  return rank === -1 ? FAMILY_ORDER.length : rank;
}

export function VisualEditor({
  principal,
  roles,
  legacy,
  onAdd,
  onRemove,
}: {
  principal: MockPrincipal;
  roles: MockRole[];
  legacy: string[];
  onAdd: (permission: string) => void;
  onRemove: (permission: string) => void;
}) {
  const byFamily = new Map<string, string[]>();
  for (const permission of [...principal.permissions].sort()) {
    const family = familyOf(permission);
    const list = byFamily.get(family);
    if (list) {
      list.push(permission);
    } else {
      byFamily.set(family, [permission]);
    }
  }
  const families = [...byFamily.keys()].sort(
    (a, b) => familyRank(a) - familyRank(b) || a.localeCompare(b),
  );

  return (
    <section className="rounded-lg border border-grayA-4 p-4 flex flex-col gap-4 min-w-0">
      <header className="flex flex-col gap-0.5">
        <h2 className="text-sm font-medium text-gray-12">Visual</h2>
        <p className="text-xs text-gray-10">
          Direct grants on this root key. Removing a chip or adding a grant commits immediately.
        </p>
      </header>

      {families.length === 0 && legacy.length === 0 ? (
        <p className="text-sm text-gray-10 rounded-md bg-grayA-2 px-3 py-2">
          No direct permissions. Everything this key can do comes from its roles.
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          {families.map((family) => (
            <div key={family} className="flex flex-col gap-1.5">
              <span className="text-xs uppercase tracking-wide text-gray-10">
                {FAMILY_LABELS[family] ?? family}
              </span>
              <div className="flex flex-wrap gap-1.5">
                {(byFamily.get(family) ?? []).map((permission) => (
                  <PermissionChip
                    key={permission}
                    value={permission}
                    onRemove={() => onRemove(permission)}
                  />
                ))}
              </div>
            </div>
          ))}
          {legacy.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <span className="text-xs uppercase tracking-wide text-gray-10">Legacy</span>
              <div className="flex flex-wrap gap-1.5">
                {legacy.map((permission) => (
                  <PermissionChip key={permission} value={permission} legacy />
                ))}
              </div>
              <p className="text-xs text-gray-9">
                Legacy tuples match by exact string and cannot be edited here.
              </p>
            </div>
          )}
        </div>
      )}

      <AddGrantRow existing={principal.permissions} onAdd={onAdd} />

      <div className="rounded-md bg-grayA-2 p-3 flex flex-col gap-2.5">
        <span className="text-xs uppercase tracking-wide text-gray-10">
          Via roles (read-only here)
        </span>
        {roles.length === 0 ? (
          <p className="text-xs text-gray-9">No roles attached to this key.</p>
        ) : (
          roles.map((role) => (
            <div key={role.id} className="flex flex-col gap-1 opacity-75">
              <div className="flex items-baseline gap-2">
                <span className="text-xs font-medium text-gray-12">{role.name}</span>
                <span className="text-xs text-gray-9">{role.description}</span>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {role.permissions.map((permission) => (
                  <PermissionChip key={permission} value={permission} />
                ))}
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}

function AddGrantRow({
  existing,
  onAdd,
}: {
  existing: string[];
  onAdd: (permission: string) => void;
}) {
  const [path, setPath] = useState("");
  const [action, setAction] = useState("");

  const pathError = path === "" ? null : validateResourcePath(path);
  const actions = path !== "" && pathError === null ? actionsForPath(path) : [];
  const selectedAction = actions.some((a) => a.action === action) ? action : "";
  const candidate =
    path !== "" && pathError === null && selectedAction !== "" ? perm(path, selectedAction) : null;
  const duplicate = candidate !== null && existing.includes(candidate);

  const selectPlaceholder =
    path === ""
      ? "Action"
      : pathError !== null
        ? "Fix the path first"
        : actions.length === 0
          ? "No known actions"
          : "Action";

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-col sm:flex-row gap-2">
        <Input
          value={path}
          onChange={(e) => setPath(e.target.value)}
          variant={pathError !== null ? "error" : "default"}
          placeholder="keyspaces/ks_payments_prod/keys/*"
          spellCheck={false}
          autoCapitalize="off"
          autoCorrect="off"
          aria-label="Resource path"
          className="font-mono"
          wrapperClassName="flex-1 min-w-0"
        />
        <div className="sm:w-52 shrink-0">
          <Select
            value={selectedAction === "" ? null : selectedAction}
            onValueChange={(value) => {
              if (typeof value === "string") {
                setAction(value);
              }
            }}
            disabled={actions.length === 0}
          >
            <SelectTrigger aria-label="Action">
              <SelectValue placeholder={selectPlaceholder} />
            </SelectTrigger>
            <SelectContent>
              {actions.map((a) => (
                <SelectItem key={a.action} value={a.action}>
                  <span className="font-mono text-xs">{a.action}</span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button
          variant="primary"
          size="lg"
          className="shrink-0"
          disabled={candidate === null || duplicate}
          onClick={() => {
            if (candidate === null || duplicate) {
              return;
            }
            onAdd(candidate);
            setPath("");
            setAction("");
          }}
        >
          Add grant
        </Button>
      </div>
      {pathError !== null && <p className="text-xs text-error-11">{pathError}</p>}
      {duplicate && candidate !== null && (
        <p className="text-xs text-warning-11">This key already has that exact grant.</p>
      )}
      {candidate !== null && !duplicate && (
        <div className="flex items-center gap-2 overflow-x-auto">
          <span className="text-xs text-gray-9 shrink-0">Will grant</span>
          <UrnText value={candidate} />
        </div>
      )}
    </div>
  );
}
