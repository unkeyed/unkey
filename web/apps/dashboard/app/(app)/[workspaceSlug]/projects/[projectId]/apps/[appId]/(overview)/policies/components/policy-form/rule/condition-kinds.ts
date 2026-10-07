import { match } from "@unkey/match";
import { IDENTIFIER_SOURCES } from "../../../policy-kinds/ratelimit/identifier-sources";
import type {
  ConditionOf,
  ConditionOperator,
  ConditionType,
  MatchConditionFormValues,
} from "./condition-schema";

type Option<T> = { value: T; label: string };

type ConditionKind<T extends ConditionType> = {
  label: string;
  operators: readonly Option<ConditionOf<T>["operator"]>[];
  name: { label: string; placeholder: string } | null;
  value: { ariaLabel: string; placeholder: string; regexPlaceholder: string | null };
  regexPrompt: string | null;
  defaults: (id: string) => ConditionOf<T>;
};

const STRING_OPERATORS = [
  { value: "exact", label: "equals" },
  { value: "prefix", label: "starts with" },
  { value: "regex", label: "matches regex" },
] as const;

const NAMED_OPERATORS = [...STRING_OPERATORS, { value: "present", label: "is present" }] as const;

const CONDITION_KINDS: { [T in ConditionType]: ConditionKind<T> } = {
  path: {
    label: "Path",
    operators: STRING_OPERATORS,
    name: null,
    value: { ariaLabel: "Path", placeholder: "/v1/checkout", regexPlaceholder: "^/v1/.*" },
    regexPrompt: "Describe the paths, e.g. all API routes under /api/v2",
    defaults: (id) => ({ id, type: "path", operator: "exact", value: "" }),
  },
  method: {
    label: "Method",
    operators: [{ value: "anyOf", label: "is any of" }],
    name: null,
    value: { ariaLabel: "Methods", placeholder: "Pick methods", regexPlaceholder: null },
    regexPrompt: null,
    defaults: (id) => ({ id, type: "method", operator: "anyOf", methods: [] }),
  },
  header: {
    label: "Header",
    operators: NAMED_OPERATORS,
    name: { label: "Header name", placeholder: "X-Tenant-Id" },
    value: { ariaLabel: "Value", placeholder: "value", regexPlaceholder: "^Bearer .+" },
    regexPrompt: "Describe the values, e.g. bearer tokens",
    defaults: (id) => ({ id, type: "header", name: "", operator: "exact", value: "" }),
  },
  queryParam: {
    label: "Query param",
    operators: NAMED_OPERATORS,
    name: { label: "Parameter name", placeholder: "debug" },
    value: { ariaLabel: "Value", placeholder: "value", regexPlaceholder: null },
    regexPrompt: "Describe the values, e.g. numeric values only",
    defaults: (id) => ({ id, type: "queryParam", name: "", operator: "exact", value: "" }),
  },
  remoteIp: {
    label: IDENTIFIER_SOURCES.remoteIp.label,
    operators: [
      { value: "in", label: "is in" },
      { value: "notIn", label: "is not in" },
    ],
    name: null,
    value: {
      ariaLabel: "IP ranges",
      placeholder: "203.0.113.0/24, 198.51.100.7",
      regexPlaceholder: null,
    },
    regexPrompt: null,
    defaults: (id) => ({ id, type: "remoteIp", operator: "in", value: "" }),
  },
};

function isConditionType(key: string): key is ConditionType {
  return Object.hasOwn(CONDITION_KINDS, key);
}

export const CONDITION_TYPES: readonly Option<ConditionType>[] = Object.keys(CONDITION_KINDS)
  .filter(isConditionType)
  .map((type) => ({ value: type, label: CONDITION_KINDS[type].label }));

export function getDefaultCondition(type: ConditionType, id?: string): MatchConditionFormValues {
  return CONDITION_KINDS[type].defaults(id ?? crypto.randomUUID());
}

type NamedCondition = ConditionOf<"header" | "queryParam">;
type RegexCondition = ConditionOf<"path" | "header" | "queryParam">;

export type ConditionNameField =
  | { type: "none" }
  | { type: "name"; label: string; placeholder: string; condition: NamedCondition };

export type OperatorControl =
  | { type: "fixed"; label: string }
  | { type: "select"; value: ConditionOperator; options: readonly Option<ConditionOperator>[] };

export type ConditionValueField =
  | { type: "none" }
  | { type: "value"; ariaLabel: string; placeholder: string };

export type ConditionRegex =
  | { type: "none" }
  | { type: "regex"; condition: RegexCondition; prompt: string; syntaxError: string | undefined };

export type ConditionRowState = {
  name: ConditionNameField;
  operator: OperatorControl;
  value: ConditionValueField;
  regex: ConditionRegex;
};

export function conditionRowState(condition: MatchConditionFormValues): ConditionRowState {
  const kind = CONDITION_KINDS[condition.type];
  return {
    name: nameField(condition),
    operator: operatorControl(condition),
    value:
      condition.operator === "present"
        ? { type: "none" }
        : {
            type: "value",
            ariaLabel: kind.value.ariaLabel,
            placeholder:
              condition.operator === "regex"
                ? (kind.value.regexPlaceholder ?? kind.value.placeholder)
                : kind.value.placeholder,
          },
    regex:
      condition.operator === "regex" && kind.regexPrompt !== null
        ? {
            type: "regex",
            condition,
            prompt: kind.regexPrompt,
            syntaxError: validateRegexSyntax(condition.value),
          }
        : { type: "none" },
  };
}

function nameField(condition: MatchConditionFormValues): ConditionNameField {
  const field = CONDITION_KINDS[condition.type].name;
  if (field === null || (condition.type !== "header" && condition.type !== "queryParam")) {
    return { type: "none" };
  }
  return { type: "name", ...field, condition };
}

function operatorControl(condition: MatchConditionFormValues): OperatorControl {
  const options: readonly Option<ConditionOperator>[] = CONDITION_KINDS[condition.type].operators;
  const [only, ...rest] = options;
  if (only && rest.length === 0) {
    return { type: "fixed", label: only.label };
  }
  return { type: "select", value: condition.operator, options };
}

export function withOperator(
  condition: MatchConditionFormValues,
  operator: ConditionOperator,
): MatchConditionFormValues {
  return match(condition)
    .returnType<MatchConditionFormValues>()
    .with({ type: "path" }, (c) => {
      const next = allowed(CONDITION_KINDS.path.operators, operator);
      return next ? { ...c, operator: next } : c;
    })
    .with({ type: "method" }, (c) => c)
    .with({ type: "header" }, { type: "queryParam" }, (c) => {
      const next = allowed(CONDITION_KINDS[c.type].operators, operator);
      return next ? { ...c, operator: next, value: next === "present" ? "" : c.value } : c;
    })
    .with({ type: "remoteIp" }, (c) => {
      const next = allowed(CONDITION_KINDS.remoteIp.operators, operator);
      return next ? { ...c, operator: next } : c;
    })
    .exhaustive();
}

function allowed<T extends ConditionOperator>(
  options: readonly Option<T>[],
  operator: ConditionOperator,
): T | null {
  return options.find((o) => o.value === operator)?.value ?? null;
}

function validateRegexSyntax(pattern: string): string | undefined {
  if (!pattern) {
    return undefined;
  }
  try {
    new RegExp(pattern);
    return undefined;
  } catch (e) {
    return e instanceof SyntaxError ? e.message : "Invalid regex pattern";
  }
}
