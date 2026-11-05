import { type Caller, type CallerEnvironment, type Callers, savedIn } from "./callers";
import type { ConnectionFormValues } from "./form";
import { type ListedConnection, type TargetRule, ruleOf, sameRule } from "./target";

export type ConnectionOp = { environment: CallerEnvironment } & (
  | { type: "create"; targetAppId: string; rule: TargetRule; name: string }
  | { type: "update"; id: string; rule: TargetRule; name?: string }
  | { type: "delete"; id: string }
);

export type SaveIntent =
  | { kind: "add" }
  | {
      kind: "edit";
      opened: { id: string; name: string; production: TargetRule; preview: TargetRule | null };
    };

type SavedRow = { id: string; name: string; rule: TargetRule };

const rowOf = (connection: ListedConnection): SavedRow => ({
  id: connection.id,
  name: connection.name,
  rule: ruleOf(connection),
});

export function renameOf(opened: { name: string } | null, name: string): string | null {
  const trimmed = name.trim();
  return opened && trimmed !== "" && trimmed !== opened.name ? trimmed : null;
}

type Naming = { targetAppId: string; name: string; rename: string | null };

function planUpsert(
  environment: CallerEnvironment,
  row: SavedRow | null,
  rule: TargetRule,
  { targetAppId, name, rename }: Naming,
): ConnectionOp | null {
  if (!row) {
    return { type: "create", environment, targetAppId, rule, name };
  }
  if (rename !== null && rename !== row.name) {
    return { type: "update", environment, id: row.id, rule, name: rename };
  }
  if (sameRule(rule, row.rule)) {
    return null;
  }
  return { type: "update", environment, id: row.id, rule };
}

function planPreview(
  caller: Caller | null,
  rule: TargetRule | null,
  openedRule: TargetRule | null,
  naming: Naming,
): ConnectionOp | null {
  if (!caller) {
    return null;
  }
  const row = savedIn(caller, naming.targetAppId);
  if (!rule) {
    return openedRule && row
      ? { type: "delete", environment: caller.environment, id: row.id }
      : null;
  }
  return planUpsert(caller.environment, row && rowOf(row), rule, naming);
}

export function planSave(
  callers: Callers,
  values: ConnectionFormValues,
  intent: SaveIntent,
): ConnectionOp[] {
  const { targetAppId, targets } = values;
  const name = values.name.trim();
  const opened = intent.kind === "edit" ? intent.opened : null;
  const naming = { targetAppId, name, rename: opened ? renameOf(opened, name) : name };
  const preview = planPreview(callers.preview, targets.preview, opened?.preview ?? null, naming);

  const { environment } = callers.production;
  const saved = savedIn(callers.production, targetAppId);
  const fallback = opened && { id: opened.id, name: opened.name, rule: opened.production };
  const production =
    planUpsert(environment, saved ? rowOf(saved) : fallback, targets.production, {
      ...naming,
      rename: name,
    }) ??
    (!saved && fallback && preview
      ? { type: "update" as const, environment, id: fallback.id, rule: targets.production }
      : null);

  return [production, preview].filter((op): op is ConnectionOp => op !== null);
}

export function removalOf(callers: Callers, targetAppId: string): ConnectionOp[] {
  return [callers.preview, callers.production].flatMap((caller) => {
    const row = savedIn(caller, targetAppId);
    return caller && row
      ? [{ type: "delete" as const, environment: caller.environment, id: row.id }]
      : [];
  });
}

export function removesPreview(ops: readonly ConnectionOp[]): boolean {
  return ops.some((op) => op.type === "delete" && op.environment.kind === "preview");
}
