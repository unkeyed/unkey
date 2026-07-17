"use client";

/**
 * Fill-the-holes dialog for one recipe: one Select per distinct hole, a
 * principal Select, and a live preview of the resolved permission strings.
 * Applying commits only the grants the principal does not already hold, then
 * keeps the dialog open in a success state.
 */

import {
  Button,
  DialogContainer,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { UrnText } from "../components/urn-display";
import { coverage, perm } from "../lib/mock-data";
import { usePermissionsLab } from "../lib/store";
import { parsePermission, permissionCovers } from "../lib/urn";
import {
  HOLE_META,
  type HoleKind,
  type HoleSelections,
  type PermissionTemplate,
  type Recipe,
  distinctHoles,
  resolveTemplatePath,
} from "./recipes";
import { HolePill, TemplateLine } from "./template-line";

interface PreviewRow {
  template: PermissionTemplate;
  /** canonical permission string once every hole is filled */
  permission: string | null;
  coverageCount: number;
  alreadyGranted: boolean;
}

interface ApplyResult {
  principalName: string;
  added: string[];
  skipped: string[];
}

function coverageLabel(count: number): string {
  return count === 1 ? "1 resource" : `${count} resources`;
}

export function ApplyRecipeDialog({ recipe, onClose }: { recipe: Recipe; onClose: () => void }) {
  const lab = usePermissionsLab();
  const [selections, setSelections] = useState<HoleSelections>({});
  const [principalID, setPrincipalID] = useState<string | null>(null);
  const [confirmText, setConfirmText] = useState("");
  const [result, setResult] = useState<ApplyResult | null>(null);

  const holes = distinctHoles(recipe);
  const principal = lab.state.principals.find((p) => p.id === principalID);
  const effectiveGrants = principal
    ? lab.effectivePermissions(principal.id).flatMap((grantString) => {
        const parsed = parsePermission(grantString);
        return parsed.ok ? [parsed.value] : [];
      })
    : [];

  const rows: PreviewRow[] = recipe.templates.map((template) => {
    const path = resolveTemplatePath(template, selections);
    if (path === null) {
      return { template, permission: null, coverageCount: 0, alreadyGranted: false };
    }
    const permission = perm(path, template.action);
    const parsed = parsePermission(permission);
    const alreadyGranted =
      parsed.ok && effectiveGrants.some((grant) => permissionCovers(grant, parsed.value));
    return { template, permission, coverageCount: coverage(path).length, alreadyGranted };
  });

  const allResolved = rows.every((row) => row.permission !== null);
  const newRows = rows.filter((row) => row.permission !== null && !row.alreadyGranted);
  const confirmed =
    recipe.confirmationPhrase === undefined || confirmText.trim() === recipe.confirmationPhrase;
  const canApply = allResolved && principal !== undefined && newRows.length > 0 && confirmed;

  const handleApply = () => {
    if (!principal) {
      return;
    }
    const added: string[] = [];
    for (const row of newRows) {
      if (row.permission !== null) {
        added.push(row.permission);
      }
    }
    if (added.length === 0) {
      return;
    }
    lab.commit(
      `Apply recipe: ${recipe.name}`,
      added.map((permission) => ({ op: "add" as const, principalID: principal.id, permission })),
    );
    const skipped: string[] = [];
    for (const row of rows) {
      if (row.permission !== null && row.alreadyGranted) {
        skipped.push(row.permission);
      }
    }
    toast.success(
      `+${added.length} permission${added.length === 1 ? "" : "s"} added to ${principal.name}`,
    );
    setResult({ principalName: principal.name, added, skipped });
  };

  return (
    <DialogContainer
      isOpen
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={result ? "Recipe applied" : recipe.name}
      subTitle={result ? undefined : recipe.tagline}
      footer={
        result ? (
          <div className="flex w-full items-center justify-end">
            <Button variant="primary" onClick={onClose}>
              Done
            </Button>
          </div>
        ) : (
          <div className="flex w-full items-center justify-end gap-3">
            <Button variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button
              variant="primary"
              color={recipe.caution ? "warning" : "default"}
              disabled={!canApply}
              onClick={handleApply}
            >
              {allResolved && principal
                ? `Grant ${newRows.length} permission${newRows.length === 1 ? "" : "s"}`
                : "Grant permissions"}
            </Button>
          </div>
        )
      }
    >
      {result ? (
        <SuccessView result={result} />
      ) : (
        <div className="flex flex-col gap-5">
          {recipe.caution && (
            <div className="rounded-lg border border-warningA-4 bg-warningA-3 p-3 text-xs leading-5 text-warning-11">
              {recipe.caution}
            </div>
          )}

          {/* biome-ignore lint/a11y/noLabelWithoutControl: wraps the Select whose native trigger button is a labelable descendant */}
          <label className="flex flex-col gap-1.5">
            <span className="text-xs font-medium text-gray-12">Grant to</span>
            <Select
              value={principalID}
              items={lab.state.principals.map((p) => ({ value: p.id, label: p.name }))}
              onValueChange={(value) => {
                if (typeof value === "string") {
                  setPrincipalID(value);
                }
              }}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select a root key" />
              </SelectTrigger>
              <SelectContent>
                {lab.state.principals.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    <span className="flex items-baseline gap-2">
                      {p.name}
                      <span className="font-mono text-[11px] text-gray-9">{p.id}</span>
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>

          {holes.map((kind) => (
            <HoleSelect
              key={kind}
              kind={kind}
              value={selections[kind] ?? null}
              onChange={(path) => setSelections((prev) => ({ ...prev, [kind]: path }))}
            />
          ))}

          <div className="flex flex-col gap-1.5">
            <span className="text-xs font-medium text-gray-12">Permissions to grant</span>
            <div className="flex flex-col gap-1.5">
              {rows.map((row, i) => (
                <div
                  key={`${i}-${row.template.action}`}
                  className={
                    row.permission === null
                      ? "flex items-center justify-between gap-3 rounded-md border border-dashed border-grayA-4 px-3 py-2 opacity-60"
                      : "flex items-center justify-between gap-3 rounded-md border border-grayA-4 bg-grayA-2 px-3 py-2"
                  }
                >
                  <span className="overflow-x-auto">
                    {row.permission === null ? (
                      <TemplateLine template={row.template} selections={selections} />
                    ) : (
                      <UrnText value={row.permission} />
                    )}
                  </span>
                  <span className="shrink-0 text-[11px] text-gray-9">
                    {row.permission === null
                      ? "waiting on placeholder"
                      : row.alreadyGranted
                        ? "already granted"
                        : coverageLabel(row.coverageCount)}
                  </span>
                </div>
              ))}
            </div>
            {!allResolved && (
              <p className="text-xs text-gray-10">
                Fill every placeholder above and the dimmed rows resolve into real grants.
              </p>
            )}
            {allResolved && principal && newRows.length === 0 && (
              <p className="text-xs text-gray-10">
                {principal.name} already holds every permission in this recipe. Nothing to grant.
              </p>
            )}
          </div>

          {recipe.confirmationPhrase !== undefined && (
            // biome-ignore lint/a11y/noLabelWithoutControl: wraps the Input component whose native input is a labelable descendant
            <label className="flex flex-col gap-1.5">
              <span className="text-xs font-medium text-gray-12">
                Type <span className="font-mono text-warning-11">{recipe.confirmationPhrase}</span>{" "}
                to confirm
              </span>
              <Input
                value={confirmText}
                onChange={(e) => setConfirmText(e.target.value)}
                placeholder={recipe.confirmationPhrase}
                variant={confirmed && confirmText.length > 0 ? "success" : "warning"}
              />
            </label>
          )}
        </div>
      )}
    </DialogContainer>
  );
}

function HoleSelect({
  kind,
  value,
  onChange,
}: {
  kind: HoleKind;
  value: string | null;
  onChange: (path: string) => void;
}) {
  const meta = HOLE_META[kind];
  return (
    // biome-ignore lint/a11y/noLabelWithoutControl: wraps the Select whose native trigger button is a labelable descendant
    <label className="flex flex-col gap-1.5">
      <span className="flex items-center gap-1.5 text-xs font-medium text-gray-12">
        {meta.selectLabel}
        <HolePill kind={kind} />
      </span>
      {meta.options.length === 0 ? (
        <p className="text-xs text-gray-10">
          No {meta.selectLabel.toLowerCase()} exists in this workspace yet.
        </p>
      ) : (
        <Select
          value={value}
          items={meta.options.map((o) => ({ value: o.path, label: o.label }))}
          onValueChange={(next) => {
            if (typeof next === "string") {
              onChange(next);
            }
          }}
        >
          <SelectTrigger>
            <SelectValue placeholder={`Select a ${meta.selectLabel.toLowerCase()}`} />
          </SelectTrigger>
          <SelectContent>
            {meta.options.map((option) => (
              <SelectItem key={option.path} value={option.path}>
                <span className="flex items-baseline gap-2">
                  {option.label}
                  <span className="font-mono text-[11px] text-gray-9">{option.path}</span>
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
    </label>
  );
}

function SuccessView({ result }: { result: ApplyResult }) {
  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-gray-11">
        <span className="font-medium text-success-11">
          {result.added.length} permission{result.added.length === 1 ? "" : "s"}
        </span>{" "}
        added to <span className="font-medium text-gray-12">{result.principalName}</span>
        {result.skipped.length > 0 && `, ${result.skipped.length} skipped as already granted`}.
      </p>
      <div className="flex flex-col gap-1.5">
        {result.added.map((permission) => (
          <div
            key={permission}
            className="flex items-center gap-2 rounded-md border border-grayA-4 bg-grayA-2 px-3 py-2"
          >
            <span className="font-mono text-xs font-semibold text-success-11">+</span>
            <span className="overflow-x-auto">
              <UrnText value={permission} />
            </span>
          </div>
        ))}
        {result.skipped.map((permission) => (
          <div
            key={permission}
            className="flex items-center justify-between gap-3 rounded-md border border-dashed border-grayA-4 px-3 py-2 opacity-60"
          >
            <span className="overflow-x-auto">
              <UrnText value={permission} />
            </span>
            <span className="shrink-0 text-[11px] text-gray-9">already granted</span>
          </div>
        ))}
      </div>
    </div>
  );
}
