import { applyVariableChanges, envVarErrorToast } from "@/lib/collections/deploy/env-vars";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useAppId, useProjectData } from "../../[appId]/(overview)/data-provider";
import { ALL_ENVIRONMENTS, targetEnvironmentIds } from "../environment-scope";
import {
  STATUS_IS_DIRTY,
  type ServerSnapshot,
  applyChange,
  hydrateTable,
  planChanges,
  resolveStatus,
  snapshotFromServer,
  visibleRowErrors,
} from "./autosave-plan";
import { useAppEnvVars } from "./use-app-env-vars";
import { useVariableRows } from "./variables-table/use-variable-rows";
import { splitVariableRows } from "./variables-table/variable-rows";

const AUTOSAVE_DELAY_MS = 600;

type Phase =
  | { kind: "loading" }
  | { kind: "unrepresentable" }
  | { kind: "editing"; server: ServerSnapshot };

export function useAutosaveEnvVars() {
  const appId = useAppId();
  const { projectId, environments, isEnvironmentsLoading } = useProjectData();
  const envVars = useAppEnvVars(projectId, appId);
  const draft = useVariableRows();
  const [environmentId, setEnvironmentId] = useState(ALL_ENVIRONMENTS);
  const [phase, setPhase] = useState<Phase>({ kind: "loading" });
  const [failedIntent, setFailedIntent] = useState<string | null>(null);
  const [tableFocused, setTableFocused] = useState(true);

  const environmentIds = environments.map((e) => e.id);
  if (
    phase.kind === "loading" &&
    envVars.status === "ready" &&
    !envVars.refreshing &&
    !isEnvironmentsLoading
  ) {
    const table = hydrateTable(envVars.data, environmentIds);
    if (table.kind === "unrepresentable") {
      setPhase({ kind: "unrepresentable" });
    } else {
      setPhase({ kind: "editing", server: snapshotFromServer(envVars.data) });
      if (table.rows.length > 0) {
        draft.replace(table.rows);
        setEnvironmentId(table.environmentId);
      }
    }
  }

  const { variables, errors } = splitVariableRows(draft.rows);
  const targetIds = targetEnvironmentIds(environmentId, environmentIds);
  const intent = JSON.stringify({ targetIds, variables });
  if (failedIntent !== null && failedIntent !== intent) {
    setFailedIntent(null);
  }
  const plan =
    phase.kind === "editing" && errors.size === 0
      ? planChanges({ server: phase.server, environmentIds, targetIds, variables })
      : [];
  const settledIntent = useDebouncedValue(intent, AUTOSAVE_DELAY_MS);
  const settledRows = useDebouncedValue(draft.rows, AUTOSAVE_DELAY_MS);

  const save = useMutation({ mutationFn: applyVariableChanges });

  useEffect(() => {
    if (
      settledIntent !== intent ||
      save.isLoading ||
      intent === failedIntent ||
      plan.length === 0
    ) {
      return;
    }
    save.mutate(
      { projectId, appId, changes: plan },
      {
        onSuccess: (results) => {
          setPhase((current) => {
            if (current.kind !== "editing") {
              return current;
            }
            const server = new Map(current.server);
            for (const { change, ok } of results) {
              const env = server.get(change.environmentId) ?? new Map<string, string>();
              server.set(change.environmentId, applyChange(env, change, ok));
            }
            return { kind: "editing", server };
          });
          const failure = results.find((r) => !r.ok);
          if (failure && !failure.ok) {
            setFailedIntent(intent);
            const { message, description } = envVarErrorToast(
              failure.error,
              "Failed to save environment variables",
            );
            toast.error(message, { description });
          }
        },
      },
    );
  }, [settledIntent, intent, failedIntent, plan, save, projectId, appId]);

  const status = resolveStatus({
    phase: phase.kind === "loading" && envVars.status === "failed" ? "load-failed" : phase.kind,
    failed: intent === failedIntent,
    pending: plan.length > 0 || save.isLoading,
    invalid: errors.size,
    count: variables.length,
  });

  return {
    draft: {
      ...draft,
      errors: visibleRowErrors({ errors, rows: draft.rows, settledRows, tableFocused }),
    },
    environmentId,
    setEnvironmentId,
    status,
    dirty: STATUS_IS_DIRTY[status.kind],
    retryLoad: () => {
      if (envVars.status === "failed") {
        envVars.retry();
      }
    },
    retrySave: () => setFailedIntent(null),
    onTableFocusChange: setTableFocused,
  };
}

function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(timer);
  }, [value, delayMs]);
  return debounced;
}
