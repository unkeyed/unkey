import { z } from "zod";
import { connectionNameSchema, takenNamesOf } from "./name";
import {
  type ListedConnection,
  type TargetRule,
  ruleOf,
  sameRule,
  targetError,
  targetRuleSchema,
} from "./target";

const formShape = z.object({
  targetAppId: z.string().min(1, "Choose an app"),
  name: z.string(),
  target: targetRuleSchema,
});

export type ConnectionFormValues = z.infer<typeof formShape>;

export type ConnectionOp =
  | { type: "create"; targetAppId: string; rule: TargetRule; name: string }
  | { type: "update"; id: string; rule: TargetRule; name?: string }
  | { type: "delete"; id: string };

function nameErrorOf(
  values: ConnectionFormValues,
  connections: readonly ListedConnection[],
): string | null {
  const parsed = connectionNameSchema.safeParse(values.name);
  if (!parsed.success) {
    return parsed.error.issues[0]?.message ?? "Enter a valid name";
  }
  if (takenNamesOf(connections, values.targetAppId).has(parsed.data)) {
    return `Another connection already uses the name ${parsed.data}.`;
  }
  return null;
}

export function connectionFormSchema(connections: readonly ListedConnection[]) {
  return formShape.superRefine((values, ctx) => {
    const nameError = nameErrorOf(values, connections);
    if (nameError) {
      ctx.addIssue({ code: "custom", path: ["name"], message: nameError });
    }
    const targetMessage = targetError(values.target);
    if (targetMessage) {
      ctx.addIssue({ code: "custom", path: ["target"], message: targetMessage });
    }
  });
}

export function formValuesOf(connection: ListedConnection): ConnectionFormValues {
  return {
    targetAppId: connection.targetAppId,
    name: connection.name,
    target: ruleOf(connection),
  };
}

export function planSave(
  before: ListedConnection | null,
  values: ConnectionFormValues,
): ConnectionOp | null {
  const name = values.name.trim();
  if (!before) {
    return { type: "create", targetAppId: values.targetAppId, rule: values.target, name };
  }
  const renamed = name !== before.name;
  if (!renamed && sameRule(values.target, ruleOf(before))) {
    return null;
  }
  return { type: "update", id: before.id, rule: values.target, ...(renamed ? { name } : {}) };
}
