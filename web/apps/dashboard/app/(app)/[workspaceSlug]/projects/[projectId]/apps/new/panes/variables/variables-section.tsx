"use client";

import { environmentsQueryFor } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider-queries";
import { collection } from "@/lib/collections";
import { listExistingKeys, setVariables } from "@/lib/collections/deploy/env-vars";
import { getErrorMessage } from "@/lib/unkey-client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { IconPlusOutline18, IconTrashOutline18 } from "@unkey/icons";
import { Button, Checkbox, Input, toast } from "@unkey/ui";
import { useId, useState } from "react";
import {
  type RowErrors,
  type VariableRow,
  conflictErrors,
  parseVariableRows,
} from "./variable-rows";

const emptyRow = (): VariableRow => ({ key: "", value: "", sensitive: false });

export function useVariableDraft(projectId: string, appId: string) {
  const [rows, setRows] = useState<VariableRow[]>([emptyRow()]);
  const [errors, setErrors] = useState<Map<number, RowErrors>>(new Map());
  const { data: environments } = useLiveQuery(environmentsQueryFor(projectId, [appId]), [
    projectId,
    appId,
  ]);
  const { data: existing } = useLiveQuery(
    (q) =>
      q
        .from({ v: collection.envVars })
        .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId))),
    [projectId, appId],
  );

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

  const save = async (): Promise<boolean> => {
    const parsed = parseVariableRows(rows);
    if (!parsed.ok) {
      setErrors(parsed.errors);
      return false;
    }
    if (parsed.variables.length === 0) {
      return true;
    }
    if (environments.length === 0) {
      toast.error("This app has no environments yet. Try again in a moment.");
      return false;
    }
    try {
      const environmentIds = environments.map((env) => env.id);
      const taken = new Set(
        (await listExistingKeys(projectId, appId, environmentIds)).map((v) => v.key),
      );
      const conflicts = conflictErrors(rows, taken);
      if (conflicts.size > 0) {
        setErrors(conflicts);
        return false;
      }
      await Promise.all(
        environmentIds.map((id) => setVariables(projectId, appId, id, parsed.variables)),
      );
      await collection.envVars.utils.refetch().catch(() => undefined);
      setRows([emptyRow()]);
      return true;
    } catch (error) {
      toast.error("Could not save the variables", { description: getErrorMessage(error) });
      return false;
    }
  };

  return {
    rows,
    errors,
    existingKeys: [...new Set(existing.map((v) => v.key))].sort(),
    hasErrors: errors.size > 0,
    update,
    remove,
    add: () => setRows((current) => [...current, emptyRow()]),
    save,
  };
}

type VariableDraft = ReturnType<typeof useVariableDraft>;

export function VariablesSection({ draft }: { draft: VariableDraft }) {
  const rowId = useId();
  return (
    <section className="flex flex-col gap-3 rounded-lg border border-grayA-4 bg-raised p-5">
      {draft.existingKeys.length > 0 ? (
        <p className="text-xs text-gray-10">
          Already set:{" "}
          <span className="font-mono text-gray-11">{draft.existingKeys.join(", ")}</span>
        </p>
      ) : null}
      <div className="flex flex-col gap-2">
        {draft.rows.map((row, index) => {
          const rowErrors = draft.errors.get(index);
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
                  onChange={(e) => draft.update(index, { key: e.target.value })}
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
                  onChange={(e) => draft.update(index, { value: e.target.value })}
                />
                <div className="flex shrink-0 items-center gap-1.5">
                  <Checkbox
                    id={`${rowId}-${index}-sensitive`}
                    checked={row.sensitive}
                    onCheckedChange={(checked) =>
                      draft.update(index, { sensitive: checked === true })
                    }
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
                  onClick={() => draft.remove(index)}
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
      <Button type="button" variant="outline" size="sm" className="self-start" onClick={draft.add}>
        <IconPlusOutline18 className="size-3" />
        Add variable
      </Button>
    </section>
  );
}
