"use client";

/**
 * Left column of the changesets concept: build a draft changeset op by op in
 * local state, review it as a diff with an approximate impact estimate, then
 * stage it into the shared lab store.
 */

import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { type ReactNode, useMemo, useState } from "react";
import { actionsForPath } from "../lib/catalog";
import { WORKSPACE_ID, coverage, perm } from "../lib/mock-data";
import { type ChangeOp, usePermissionsLab } from "../lib/store";
import { parsePermission, validateResourcePath } from "../lib/urn";
import { DiffLine } from "./diff-line";

type OpKind = "grant" | "revoke" | "attach_role" | "detach_role";

const OP_KIND_ITEMS: { value: OpKind; label: string }[] = [
  { value: "grant", label: "Add permission" },
  { value: "revoke", label: "Remove permission" },
  { value: "attach_role", label: "Add role" },
  { value: "detach_role", label: "Remove role" },
];

function isOpKind(value: string): value is OpKind {
  return OP_KIND_ITEMS.some((item) => item.value === value);
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    // biome-ignore lint/a11y/noLabelWithoutControl: the control is a composed component in children; its native element gets implicit label association at runtime
    <label className="flex flex-col gap-1.5">
      <span className="text-xs font-medium text-gray-11">{label}</span>
      {children}
    </label>
  );
}

function plural(count: number, singular: string, pluralForm: string): string {
  return `${count} ${count === 1 ? singular : pluralForm}`;
}

export function Composer() {
  const lab = usePermissionsLab();

  const [title, setTitle] = useState("");
  const [ops, setOps] = useState<ChangeOp[]>([]);
  const [principalID, setPrincipalID] = useState<string | null>(null);
  const [kind, setKind] = useState<OpKind>("grant");
  const [resourcePath, setResourcePath] = useState("");
  const [action, setAction] = useState<string | null>(null);
  const [removePermission, setRemovePermission] = useState<string | null>(null);
  const [roleID, setRoleID] = useState<string | null>(null);

  const principal = lab.state.principals.find((p) => p.id === principalID);

  const pathError = resourcePath === "" ? null : validateResourcePath(resourcePath);
  const availableActions =
    resourcePath !== "" && pathError === null ? actionsForPath(resourcePath) : [];
  // The action select can hold a value from a previously typed path; only a
  // value valid for the current path counts.
  const validAction =
    action !== null && availableActions.some((a) => a.action === action) ? action : null;
  const matchCount =
    resourcePath !== "" && pathError === null ? coverage(resourcePath).length : null;

  const draftRemovals = useMemo(
    () =>
      new Set(
        ops.flatMap((op) =>
          "permission" in op && op.op === "remove" ? [`${op.principalID} ${op.permission}`] : [],
        ),
      ),
    [ops],
  );
  const draftRoleOps = (opName: "add_role" | "remove_role") =>
    new Set(
      ops.flatMap((op) =>
        "roleID" in op && op.op === opName && op.principalID === principalID ? [op.roleID] : [],
      ),
    );

  const removablePermissions = principal
    ? principal.permissions.filter((p) => !draftRemovals.has(`${principal.id} ${p}`))
    : [];
  const attachableRoles = principal
    ? lab.state.roles.filter(
        (r) => !principal.roles.includes(r.id) && !draftRoleOps("add_role").has(r.id),
      )
    : [];
  const detachableRoles = principal
    ? lab.state.roles.filter(
        (r) => principal.roles.includes(r.id) && !draftRoleOps("remove_role").has(r.id),
      )
    : [];

  function buildOp(): ChangeOp | null {
    if (!principalID) {
      return null;
    }
    switch (kind) {
      case "grant": {
        if (resourcePath === "" || pathError !== null || validAction === null) {
          return null;
        }
        const permission = perm(resourcePath, validAction);
        return parsePermission(permission).ok ? { op: "add", principalID, permission } : null;
      }
      case "revoke":
        return removePermission !== null
          ? { op: "remove", principalID, permission: removePermission }
          : null;
      case "attach_role":
        return roleID !== null && attachableRoles.some((r) => r.id === roleID)
          ? { op: "add_role", principalID, roleID }
          : null;
      case "detach_role":
        return roleID !== null && detachableRoles.some((r) => r.id === roleID)
          ? { op: "remove_role", principalID, roleID }
          : null;
    }
  }

  const pendingOp = buildOp();

  function addToDraft() {
    const op = buildOp();
    if (!op) {
      return;
    }
    if (ops.some((existing) => JSON.stringify(existing) === JSON.stringify(op))) {
      toast.error("This change is already in the draft");
      return;
    }
    setOps([...ops, op]);
    setResourcePath("");
    setAction(null);
    setRemovePermission(null);
    setRoleID(null);
  }

  function stageDraft() {
    const trimmed = title.trim();
    if (trimmed === "" || ops.length === 0) {
      return;
    }
    lab.stage(trimmed, ops);
    toast.success(`Staged "${trimmed}"`);
    setOps([]);
    setTitle("");
  }

  const impact = useMemo(() => {
    const principalIDs = new Set(ops.map((op) => op.principalID));
    // Union of concrete resources the added patterns cover, minus what each
    // principal can already reach for that action. Counted per resource so
    // overlapping grants are not double counted.
    const newlyReachable = new Set<string>();
    for (const op of ops) {
      if (!("permission" in op) || op.op !== "add") {
        continue;
      }
      const parsed = parsePermission(op.permission);
      if (!parsed.ok) {
        continue;
      }
      const { urn, action: grantedAction } = parsed.value;
      for (const resource of coverage(urn.resource)) {
        const request = {
          urn: { workspaceID: WORKSPACE_ID, resource: resource.path },
          action: grantedAction,
        };
        if (!lab.can(op.principalID, request)) {
          newlyReachable.add(resource.path);
        }
      }
    }
    return { principals: principalIDs.size, newlyReachable: newlyReachable.size };
  }, [ops, lab]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-4 rounded-lg border border-grayA-4 p-4">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Principal">
            <Select
              value={principalID}
              items={lab.state.principals.map((p) => ({ value: p.id, label: p.name }))}
              onValueChange={(value) => {
                if (typeof value === "string") {
                  setPrincipalID(value);
                  setRemovePermission(null);
                  setRoleID(null);
                }
              }}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select principal" />
              </SelectTrigger>
              <SelectContent>
                {lab.state.principals.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field label="Change">
            <Select
              value={kind}
              items={OP_KIND_ITEMS}
              onValueChange={(value) => {
                if (typeof value === "string" && isOpKind(value)) {
                  setKind(value);
                  setRemovePermission(null);
                  setRoleID(null);
                }
              }}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {OP_KIND_ITEMS.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
        </div>

        {kind === "grant" && (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div className="flex flex-col gap-1">
              <Field label="Resource path">
                <Input
                  value={resourcePath}
                  onChange={(e) => setResourcePath(e.target.value)}
                  placeholder="keyspaces/ks_payments_prod/keys/*"
                  className="font-mono text-xs"
                  variant={pathError !== null ? "error" : "default"}
                  disabled={!principal}
                />
              </Field>
              {pathError !== null && <span className="text-xs text-error-11">{pathError}</span>}
              {pathError === null && matchCount !== null && (
                <span className="text-xs text-gray-10">
                  Matches {plural(matchCount, "existing resource", "existing resources")}
                </span>
              )}
            </div>
            <div className="flex flex-col gap-1">
              <Field label="Action">
                <Select
                  value={validAction}
                  items={availableActions.map((a) => ({ value: a.action, label: a.action }))}
                  onValueChange={(value) => {
                    if (typeof value === "string") {
                      setAction(value);
                    }
                  }}
                >
                  <SelectTrigger
                    className="font-mono text-xs"
                    disabled={!principal || availableActions.length === 0}
                  >
                    <SelectValue placeholder="Select action" />
                  </SelectTrigger>
                  <SelectContent>
                    {availableActions.map((a) => (
                      <SelectItem key={a.action} value={a.action}>
                        <span className="flex flex-col">
                          <span className="font-mono text-xs">{a.action}</span>
                          <span className="text-xs text-gray-10">{a.description}</span>
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
              {resourcePath !== "" && pathError === null && availableActions.length === 0 && (
                <span className="text-xs text-warning-11">No known actions for this path</span>
              )}
            </div>
          </div>
        )}

        {kind === "revoke" &&
          (removablePermissions.length > 0 ? (
            <Field label="Direct grant to remove">
              <Select
                value={removePermission}
                items={removablePermissions.map((p) => ({ value: p, label: p }))}
                onValueChange={(value) => {
                  if (typeof value === "string") {
                    setRemovePermission(value);
                  }
                }}
              >
                <SelectTrigger className="font-mono text-xs" disabled={!principal}>
                  <SelectValue placeholder="Select permission" />
                </SelectTrigger>
                <SelectContent>
                  {removablePermissions.map((p) => (
                    <SelectItem key={p} value={p}>
                      <span className="font-mono text-xs">{p}</span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
          ) : (
            <p className="text-sm text-gray-10">
              {principal
                ? `${principal.name} has no direct grants left to remove.`
                : "Select a principal to see its direct grants."}
            </p>
          ))}

        {(kind === "attach_role" || kind === "detach_role") &&
          (() => {
            const options = kind === "attach_role" ? attachableRoles : detachableRoles;
            if (!principal) {
              return <p className="text-sm text-gray-10">Select a principal to manage roles.</p>;
            }
            if (options.length === 0) {
              return (
                <p className="text-sm text-gray-10">
                  {kind === "attach_role"
                    ? `${principal.name} already has every role.`
                    : `${principal.name} has no roles left to remove.`}
                </p>
              );
            }
            return (
              <Field label={kind === "attach_role" ? "Role to add" : "Role to remove"}>
                <Select
                  value={roleID}
                  items={options.map((r) => ({ value: r.id, label: r.name }))}
                  onValueChange={(value) => {
                    if (typeof value === "string") {
                      setRoleID(value);
                    }
                  }}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select role" />
                  </SelectTrigger>
                  <SelectContent>
                    {options.map((r) => (
                      <SelectItem key={r.id} value={r.id}>
                        <span className="flex flex-col">
                          <span>{r.name}</span>
                          <span className="text-xs text-gray-10">{r.description}</span>
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            );
          })()}

        <div className="flex justify-end">
          <Button variant="outline" disabled={pendingOp === null} onClick={addToDraft}>
            Add to draft
          </Button>
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <span className="text-xs uppercase tracking-wide text-gray-10">Draft changes</span>
        {ops.length === 0 ? (
          <p className="rounded-md border border-dashed border-grayA-4 px-3 py-4 text-center text-sm text-gray-10">
            No changes yet, add one above.
          </p>
        ) : (
          <div className="flex flex-col gap-1.5">
            {ops.map((op) => (
              <DiffLine
                key={JSON.stringify(op)}
                op={op}
                onRemove={() => setOps(ops.filter((o) => JSON.stringify(o) !== JSON.stringify(op)))}
              />
            ))}
          </div>
        )}
      </div>

      {ops.length > 0 && (
        <div className="flex flex-col gap-1 rounded-lg border border-grayA-4 bg-grayA-2 p-3">
          <span className="text-xs uppercase tracking-wide text-gray-10">Approximate impact</span>
          <p className="text-sm text-gray-11">
            {plural(impact.principals, "principal", "principals")} affected.{" "}
            {plural(impact.newlyReachable, "resource becomes", "resources become")} newly reachable
            through the added grants.
          </p>
          <p className="text-xs text-gray-10">
            Counted against the resources that exist in this workspace today; wildcards also cover
            resources created later.
          </p>
        </div>
      )}

      <div className="flex items-end gap-2">
        <div className="flex flex-1 flex-col gap-1.5">
          <Field label="Changeset title">
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Rotate payments access"
            />
          </Field>
        </div>
        <Button
          className="h-9"
          disabled={title.trim() === "" || ops.length === 0}
          onClick={stageDraft}
        >
          Stage changeset
        </Button>
      </div>
    </div>
  );
}
