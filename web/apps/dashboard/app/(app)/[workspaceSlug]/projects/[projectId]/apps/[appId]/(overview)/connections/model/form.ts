import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import { z } from "zod";
import { type Callers, callerList, savedIn } from "./callers";
import { connectionNameSchema, takenNamesOf } from "./name";
import { type ConnectionOp, type SaveIntent, planSave } from "./plan";
import {
  type ListedConnection,
  type TargetRule,
  isProduction,
  ruleOf,
  targetError,
  targetRuleSchema,
} from "./target";

const formShape = z.object({
  targetAppId: z.string().min(1, "Choose an app"),
  name: z.string(),
  targets: z.object({
    production: targetRuleSchema,
    preview: targetRuleSchema.nullable(),
  }),
});

export type ConnectionFormValues = z.infer<typeof formShape>;

const AUTOMATIC: TargetRule = { targetType: "automatic" };

function setsName(op: ConnectionOp): boolean {
  return op.type === "create" || (op.type === "update" && op.name !== undefined);
}

export function takenNames(
  callers: Callers,
  targetAppId: string,
  ops: readonly ConnectionOp[],
): ReadonlySet<string> {
  const naming = new Set(ops.filter(setsName).map((op) => op.environment.kind));
  const inUse = callerList(callers)
    .filter(({ environment }) => isProduction(environment) || naming.has(environment.kind))
    .flatMap(({ connections }) => connections);
  return takenNamesOf(inUse, targetAppId);
}

function nameErrorOf(name: string, taken: ReadonlySet<string>): string | null {
  const parsed = connectionNameSchema.safeParse(name);
  if (!parsed.success) {
    return parsed.error.issues[0]?.message ?? "Enter a valid name";
  }
  if (taken.has(parsed.data)) {
    return `Another connection already uses the name ${parsed.data}.`;
  }
  return null;
}

export function connectionFormSchema(callers: Callers, intent: SaveIntent) {
  return formShape.superRefine((values, ctx) => {
    const taken = takenNames(callers, values.targetAppId, planSave(callers, values, intent));
    const nameError = nameErrorOf(values.name, taken);
    if (nameError) {
      ctx.addIssue({ code: "custom", path: ["name"], message: nameError });
    }
    for (const kind of ENVIRONMENT_KINDS) {
      const rule = values.targets[kind];
      const message = rule ? targetError(rule) : null;
      if (message) {
        ctx.addIssue({ code: "custom", path: ["targets", kind], message });
      }
    }
  });
}

const EMPTY_VALUES: ConnectionFormValues = {
  targetAppId: "",
  name: "",
  targets: { production: AUTOMATIC, preview: null },
};

export function openForm(
  connection: ListedConnection | null,
  callers: Callers,
): { values: ConnectionFormValues; intent: SaveIntent } {
  if (!connection) {
    return { values: EMPTY_VALUES, intent: { kind: "add" } };
  }
  const companion = savedIn(callers.preview, connection.targetAppId);
  const targets = {
    production: ruleOf(connection),
    preview: companion ? ruleOf(companion) : null,
  };
  return {
    values: { targetAppId: connection.targetAppId, name: connection.name, targets },
    intent: {
      kind: "edit",
      opened: { id: connection.id, name: connection.name, ...targets },
    },
  };
}

export function targetsForApp(
  callers: Callers,
  current: ConnectionFormValues,
  targetAppId: string,
): ConnectionFormValues["targets"] {
  const production = savedIn(callers.production, targetAppId);
  const preview = savedIn(callers.preview, targetAppId);
  const carried =
    current.targets.preview !== null && !savedIn(callers.preview, current.targetAppId)
      ? AUTOMATIC
      : null;
  return {
    production: production ? ruleOf(production) : AUTOMATIC,
    preview: preview ? ruleOf(preview) : carried,
  };
}
