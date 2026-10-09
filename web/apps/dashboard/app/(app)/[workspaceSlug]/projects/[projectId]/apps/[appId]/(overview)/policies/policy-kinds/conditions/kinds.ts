import type { Router } from "@/lib/trpc/routers";
import type { inferRouterInputs } from "@trpc/server";
import type {
  ConditionOf,
  ConditionOperator,
  ConditionType,
  MatchConditionFormValues,
} from "./schema";

type Option<T> = { value: T; label: string };

export type RegexConditionType =
  inferRouterInputs<Router>["deploy"]["environmentSettings"]["policies"]["generateRegex"]["conditionType"];

type NamedCondition = Extract<MatchConditionFormValues, { name: string }>;
type TextCondition = Extract<MatchConditionFormValues, { value: string }>;
type MethodCondition = Extract<MatchConditionFormValues, { methods: unknown }>;
type RegexCondition = ConditionOf<RegexConditionType>;

type ConditionKind<T extends ConditionType> = {
  label: string;
  operators: readonly Option<ConditionOf<T>["operator"]>[];
  name: ConditionOf<T> extends NamedCondition ? { label: string; placeholder: string } : null;
  value: { ariaLabel: string; placeholder: string; regexPlaceholder: string | null };
  regexPrompt: T extends RegexConditionType ? string : null;
  defaults: () => ConditionOf<T>;
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
    defaults: () => ({ type: "path", operator: "exact", value: "" }),
  },
  method: {
    label: "Method",
    operators: [{ value: "anyOf", label: "is any of" }],
    name: null,
    value: { ariaLabel: "Methods", placeholder: "Pick methods", regexPlaceholder: null },
    regexPrompt: null,
    defaults: () => ({ type: "method", operator: "anyOf", methods: [] }),
  },
  header: {
    label: "Header",
    operators: NAMED_OPERATORS,
    name: { label: "Header name", placeholder: "X-Tenant-Id" },
    value: { ariaLabel: "Value", placeholder: "value", regexPlaceholder: "^Bearer .+" },
    regexPrompt: "Describe the values, e.g. bearer tokens",
    defaults: () => ({ type: "header", name: "", operator: "exact", value: "" }),
  },
  queryParam: {
    label: "Query param",
    operators: NAMED_OPERATORS,
    name: { label: "Parameter name", placeholder: "debug" },
    value: { ariaLabel: "Value", placeholder: "value", regexPlaceholder: null },
    regexPrompt: "Describe the values, e.g. numeric values only",
    defaults: () => ({ type: "queryParam", name: "", operator: "exact", value: "" }),
  },
  remoteIp: {
    label: "Client IP",
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
    defaults: () => ({ type: "remoteIp", operator: "in", value: "" }),
  },
};

function isConditionType(key: string): key is ConditionType {
  return Object.hasOwn(CONDITION_KINDS, key);
}

export const CONDITION_TYPES: readonly Option<ConditionType>[] = Object.keys(CONDITION_KINDS)
  .filter(isConditionType)
  .map((type) => ({ value: type, label: CONDITION_KINDS[type].label }));

export function getDefaultCondition(type: ConditionType): MatchConditionFormValues {
  return CONDITION_KINDS[type].defaults();
}

export type ConditionNameField =
  | { type: "none" }
  | { type: "name"; label: string; placeholder: string; condition: NamedCondition };

export type OperatorControl =
  | { type: "fixed"; label: string }
  | { type: "select"; value: ConditionOperator; options: readonly Option<ConditionOperator>[] };

export type ConditionValueField =
  | { type: "none" }
  | { type: "text"; ariaLabel: string; placeholder: string; condition: TextCondition }
  | { type: "methods"; ariaLabel: string; placeholder: string; condition: MethodCondition };

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
  return {
    name: nameField(condition),
    operator: operatorControl(condition),
    value: valueField(condition),
    regex: regexField(condition),
  };
}

function nameField(condition: MatchConditionFormValues): ConditionNameField {
  if (!("name" in condition)) {
    return { type: "none" };
  }
  return { type: "name", ...CONDITION_KINDS[condition.type].name, condition };
}

function operatorControl(condition: MatchConditionFormValues): OperatorControl {
  const options: readonly Option<ConditionOperator>[] = CONDITION_KINDS[condition.type].operators;
  const [only, ...rest] = options;
  if (only && rest.length === 0) {
    return { type: "fixed", label: only.label };
  }
  return { type: "select", value: condition.operator, options };
}

function valueField(condition: MatchConditionFormValues): ConditionValueField {
  if (condition.operator === "present") {
    return { type: "none" };
  }
  const { ariaLabel, placeholder, regexPlaceholder } = CONDITION_KINDS[condition.type].value;
  const copy = {
    ariaLabel,
    placeholder: condition.operator === "regex" ? (regexPlaceholder ?? placeholder) : placeholder,
  };
  return "methods" in condition
    ? { type: "methods", ...copy, condition }
    : { type: "text", ...copy, condition };
}

function hasRegexPrompt(condition: MatchConditionFormValues): condition is RegexCondition {
  return CONDITION_KINDS[condition.type].regexPrompt !== null;
}

function regexField(condition: MatchConditionFormValues): ConditionRegex {
  if (condition.operator !== "regex" || !hasRegexPrompt(condition)) {
    return { type: "none" };
  }
  return {
    type: "regex",
    condition,
    prompt: CONDITION_KINDS[condition.type].regexPrompt,
    syntaxError: validateRegexSyntax(condition.value),
  };
}

type AnyOperatorCondition = {
  [T in ConditionType]: Omit<ConditionOf<T>, "operator"> & { operator: ConditionOperator };
}[ConditionType];

function hasAllowedOperator(c: AnyOperatorCondition): c is MatchConditionFormValues {
  const options: readonly Option<ConditionOperator>[] = CONDITION_KINDS[c.type].operators;
  return options.some((o) => o.value === c.operator);
}

export function withOperator(
  condition: MatchConditionFormValues,
  operator: ConditionOperator,
): MatchConditionFormValues {
  const next: AnyOperatorCondition = { ...condition, operator };
  if (!hasAllowedOperator(next)) {
    return condition;
  }
  return next.operator === "present" ? { ...next, value: "" } : next;
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
