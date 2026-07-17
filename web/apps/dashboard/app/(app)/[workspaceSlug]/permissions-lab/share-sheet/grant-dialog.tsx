"use client";

/**
 * "Grant access" dialog: pick a principal, pick actions for the selected
 * resource, choose the scope (only this resource, or this and everything
 * under it), and preview the exact permission strings before committing one
 * add op per action.
 */

import {
  Button,
  Checkbox,
  DialogContainer,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  cn,
  toast,
} from "@unkey/ui";
import { useEffect, useMemo, useState } from "react";
import { UrnText } from "../components/urn-display";
import { actionsForPath } from "../lib/catalog";
import { ALL_RESOURCES, type ConcreteResource, coverage, perm } from "../lib/mock-data";
import { usePermissionsLab } from "../lib/store";

type Scope = "exact" | "descendants";

function targetPathFor(resource: ConcreteResource, scope: Scope): string {
  return scope === "descendants" ? `${resource.path}/**` : resource.path;
}

export function GrantDialog({
  resource,
  isOpen,
  onOpenChange,
}: {
  resource: ConcreteResource;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const lab = usePermissionsLab();

  const [principalID, setPrincipalID] = useState<string | null>(null);
  const [scope, setScope] = useState<Scope>("exact");
  const [selectedActions, setSelectedActions] = useState<Set<string>>(new Set());

  // Fresh form every time the sheet opens.
  useEffect(() => {
    if (isOpen) {
      setPrincipalID(null);
      setScope("exact");
      setSelectedActions(new Set());
    }
  }, [isOpen]);

  const principals = lab.state.principals;
  const principal = principals.find((p) => p.id === principalID) ?? null;

  const hasChildren = useMemo(
    () => ALL_RESOURCES.some((r) => r.path.startsWith(`${resource.path}/`)),
    [resource.path],
  );
  const descendantCount = useMemo(
    () => (hasChildren ? coverage(`${resource.path}/**`).length : 0),
    [hasChildren, resource.path],
  );

  const targetPath = targetPathFor(resource, scope);
  const actions = useMemo(() => actionsForPath(targetPath), [targetPath]);

  const effective = useMemo(
    () => new Set(principal ? lab.effectivePermissions(principal.id) : []),
    [principal, lab],
  );

  const changeScope = (next: Scope) => {
    setScope(next);
    const allowed = new Set(actionsForPath(targetPathFor(resource, next)).map((a) => a.action));
    setSelectedActions((prev) => new Set([...prev].filter((a) => allowed.has(a))));
  };

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

  const previewActions = [...selectedActions].sort();
  const newPermissions = previewActions
    .map((action) => perm(targetPath, action))
    .filter((permission) => !effective.has(permission));

  const grant = () => {
    if (!principal || newPermissions.length === 0) {
      return;
    }
    lab.commit(
      `Share ${resource.label} with ${principal.name}`,
      newPermissions.map((permission) => ({
        op: "add" as const,
        principalID: principal.id,
        permission,
      })),
    );
    toast.success(`Shared ${resource.label} with ${principal.name}`, {
      description: `${newPermissions.length} permission${newPermissions.length === 1 ? "" : "s"} granted`,
    });
    onOpenChange(false);
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      title={`Share ${resource.label}`}
      subTitle="Grant a principal access to this resource"
      footer={
        <div className="flex w-full items-center justify-end gap-3">
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="primary"
            disabled={!principal || newPermissions.length === 0}
            onClick={grant}
          >
            Grant access
          </Button>
        </div>
      }
    >
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-1.5">
          <span className="text-xs font-medium text-gray-11">Principal</span>
          <Select value={principalID} onValueChange={(value) => setPrincipalID(value)}>
            <SelectTrigger aria-label="Principal">
              {principal ? (
                <span className="text-gray-12">{principal.name}</span>
              ) : (
                <span className="text-grayA-8">Choose a principal</span>
              )}
            </SelectTrigger>
            <SelectContent>
              {principals.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  <span className="flex items-baseline gap-2">
                    <span>{p.name}</span>
                    <span className="font-mono text-[11px] text-gray-9">{p.id}</span>
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {hasChildren && (
          <div className="flex flex-col gap-1.5">
            <span className="text-xs font-medium text-gray-11">Scope</span>
            <div
              className="grid grid-cols-1 sm:grid-cols-2 gap-2"
              role="radiogroup"
              aria-label="Scope"
            >
              {(
                [
                  {
                    value: "exact" as const,
                    title: "Only this resource",
                    detail: `Grants apply to ${resource.label} alone`,
                  },
                  {
                    value: "descendants" as const,
                    title: "This and everything under it",
                    detail: `Appends /** and covers ${descendantCount} resources today, plus anything created later`,
                  },
                ] satisfies { value: Scope; title: string; detail: string }[]
              ).map((option) => {
                const active = scope === option.value;
                return (
                  <button
                    key={option.value}
                    type="button"
                    // biome-ignore lint/a11y/useSemanticElements: radio-card option; a native radio input cannot contain this rich layout
                    role="radio"
                    aria-checked={active}
                    onClick={() => changeScope(option.value)}
                    className={cn(
                      "flex items-start gap-2.5 rounded-lg border p-3 text-left transition-colors",
                      active
                        ? "border-grayA-8 bg-grayA-2"
                        : "border-grayA-4 hover:border-grayA-6 hover:bg-grayA-2",
                    )}
                  >
                    <span
                      aria-hidden="true"
                      className={cn(
                        "mt-0.5 flex size-3.5 shrink-0 items-center justify-center rounded-full border",
                        active ? "border-accent-12" : "border-grayA-6",
                      )}
                    >
                      {active && <span className="size-1.5 rounded-full bg-accent-12" />}
                    </span>
                    <span className="flex flex-col gap-0.5">
                      <span className="text-[13px] font-medium text-gray-12">{option.title}</span>
                      <span className="text-xs text-gray-11">{option.detail}</span>
                    </span>
                  </button>
                );
              })}
            </div>
          </div>
        )}

        <div className="flex flex-col gap-1.5">
          <span className="text-xs font-medium text-gray-11">Actions</span>
          {actions.length === 0 ? (
            <p className="text-xs text-gray-10">No actions are defined for this resource type.</p>
          ) : (
            <div className="flex flex-col rounded-lg border border-grayA-4 divide-y divide-grayA-3">
              {actions.map((action) => {
                const alreadyGranted = effective.has(perm(targetPath, action.action));
                return (
                  // biome-ignore lint/a11y/noLabelWithoutControl: wraps the Checkbox whose native labelable element is a descendant
                  <label
                    key={action.action}
                    className={cn(
                      "flex items-start gap-2.5 px-3 py-2",
                      alreadyGranted ? "opacity-60" : "cursor-pointer hover:bg-grayA-2",
                    )}
                  >
                    <Checkbox
                      className="mt-0.5"
                      checked={alreadyGranted || selectedActions.has(action.action)}
                      disabled={alreadyGranted}
                      onCheckedChange={(checked) => toggleAction(action.action, checked)}
                    />
                    <span className="flex min-w-0 flex-col gap-0.5">
                      <span className="flex items-baseline gap-2">
                        <span className="font-mono text-xs text-gray-12">{action.action}</span>
                        {alreadyGranted && (
                          <span className="text-[10px] uppercase tracking-wide text-gray-9">
                            already granted
                          </span>
                        )}
                      </span>
                      <span className="text-xs text-gray-11">{action.description}</span>
                    </span>
                  </label>
                );
              })}
            </div>
          )}
        </div>

        <div className="flex flex-col gap-1.5">
          <span className="text-xs font-medium text-gray-11">
            {principal ? `This grants ${principal.name}` : "This grants the selected principal"}
          </span>
          <div className="rounded-lg border border-grayA-4 bg-grayA-2 p-3">
            {previewActions.length === 0 ? (
              <p className="text-xs text-gray-10">Select at least one action to see the grants.</p>
            ) : (
              <div className="flex flex-col gap-1 overflow-x-auto">
                {previewActions.map((action) => (
                  <UrnText key={action} value={perm(targetPath, action)} />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </DialogContainer>
  );
}
