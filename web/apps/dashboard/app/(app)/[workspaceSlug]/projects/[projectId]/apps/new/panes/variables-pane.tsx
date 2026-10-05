"use client";

import { environmentsQueryFor } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider-queries";
import { collection } from "@/lib/collections";
import { listExistingKeys, setVariables } from "@/lib/collections/deploy/env-vars";
import { getErrorMessage } from "@/lib/unkey-client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { IconPlusOutline18, IconTrashOutline18 } from "@unkey/icons";
import { Button, Checkbox, Input, toast } from "@unkey/ui";
import { useId, useState } from "react";
import { useNewAppFlow } from "../flow";
import { PaneActions } from "./pane-actions";
import {
  type RowErrors,
  type VariableRow,
  conflictErrors,
  parseVariableRows,
} from "./variables/variable-rows";

const emptyRow = (): VariableRow => ({ key: "", value: "", sensitive: false });

export function VariablesPane({ appId }: { appId: string }) {
  const { projectId, dispatch } = useNewAppFlow();
  const onContinue = () => dispatch({ type: "next" });
  const [rows, setRows] = useState<VariableRow[]>([emptyRow()]);
  const [errors, setErrors] = useState<Map<number, RowErrors>>(new Map());
  const [saving, setSaving] = useState(false);
  const rowId = useId();

  const { data: environments, isLoading: environmentsLoading } = useLiveQuery(
    environmentsQueryFor(projectId, [appId]),
    [projectId, appId],
  );
  const { data: existing } = useLiveQuery(
    (q) =>
      q
        .from({ v: collection.envVars })
        .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId))),
    [projectId, appId],
  );
  const existingKeys = [...new Set(existing.map((v) => v.key))].sort();

  const update = (index: number, patch: Partial<VariableRow>) => {
    setRows((current) => current.map((row, i) => (i === index ? { ...row, ...patch } : row)));
    setErrors((current) => {
      const next = new Map(current);
      next.delete(index);
      return next;
    });
  };

  const remove = (index: number) => {
    setRows((current) => {
      const next = current.filter((_, i) => i !== index);
      return next.length > 0 ? next : [emptyRow()];
    });
    setErrors(new Map());
  };

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    const parsed = parseVariableRows(rows);
    if (!parsed.ok) {
      setErrors(parsed.errors);
      return;
    }
    if (parsed.variables.length === 0) {
      onContinue();
      return;
    }
    if (environments.length === 0) {
      toast.error("This app has no environments yet. Try again in a moment.");
      return;
    }
    setSaving(true);
    try {
      const environmentIds = environments.map((env) => env.id);
      const existingKeys = new Set(
        (await listExistingKeys(projectId, appId, environmentIds)).map((v) => v.key),
      );
      const conflicts = conflictErrors(rows, existingKeys);
      if (conflicts.size > 0) {
        setErrors(conflicts);
        return;
      }
      await Promise.all(
        environmentIds.map((id) => setVariables(projectId, appId, id, parsed.variables)),
      );
      await collection.envVars.utils.refetch().catch(() => undefined);
      setRows([emptyRow()]);
      onContinue();
    } catch (error) {
      toast.error("Could not save the variables", { description: getErrorMessage(error) });
    } finally {
      setSaving(false);
    }
  };

  const formId = useId();
  return (
    <form id={formId} className="flex flex-1 flex-col gap-3" onSubmit={submit}>
      {existingKeys.length > 0 ? (
        <p className="text-xs text-gray-10">
          Already set: <span className="font-mono text-gray-11">{existingKeys.join(", ")}</span>
        </p>
      ) : null}
      <div className="flex flex-col gap-2">
        {rows.map((row, index) => {
          const rowErrors = errors.get(index);
          return (
            // biome-ignore lint/suspicious/noArrayIndexKey: rows have no identity besides their position
            <div key={index} className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                <Input
                  aria-label="Name"
                  placeholder="KEY"
                  spellCheck={false}
                  data-1p-ignore
                  autoComplete="off"
                  className="font-mono text-xs"
                  variant={rowErrors?.key ? "error" : "default"}
                  value={row.key}
                  onChange={(e) => update(index, { key: e.target.value })}
                />
                <Input
                  aria-label="Value"
                  placeholder="value"
                  spellCheck={false}
                  data-1p-ignore
                  autoComplete="off"
                  className="font-mono text-xs"
                  variant={rowErrors?.value ? "error" : "default"}
                  value={row.value}
                  onChange={(e) => update(index, { value: e.target.value })}
                />
                <div className="flex shrink-0 items-center gap-1.5">
                  <Checkbox
                    id={`${rowId}-${index}-sensitive`}
                    checked={row.sensitive}
                    onCheckedChange={(checked) => update(index, { sensitive: checked === true })}
                  />
                  <label htmlFor={`${rowId}-${index}-sensitive`} className="text-xs text-gray-11">
                    Sensitive
                  </label>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label="Remove variable"
                  onClick={() => remove(index)}
                >
                  <IconTrashOutline18 className="size-3.5 text-gray-9" />
                </Button>
              </div>
              {rowErrors ? (
                <p className="text-xs text-error-11">{rowErrors.key ?? rowErrors.value}</p>
              ) : null}
            </div>
          );
        })}
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="self-start"
        onClick={() => setRows((current) => [...current, emptyRow()])}
      >
        <IconPlusOutline18 className="size-3" />
        Add variable
      </Button>
      <PaneActions>
        <Button
          type="submit"
          form={formId}
          variant="primary"
          size="sm"
          className="px-3"
          loading={saving}
          disabled={saving || environmentsLoading}
        >
          Continue
        </Button>
      </PaneActions>
    </form>
  );
}
