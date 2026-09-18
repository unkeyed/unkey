import type { Example, SearchSpec } from "./spec";

const SHARED_RULES = [
  "Read the whole query before answering. A query may name several fields at once.",
  "Leave a field out entirely when the query says nothing about it. An empty answer is better than a guessed one.",
  'A word that names a field rather than a value is not a value. In "keys whose name contains staging" the only value is staging.',
  "Prefer what the query says outright over what it implies.",
  "Prefer the more specific reading when two are available.",
];

const SHARED_PRIORITIES = [
  "An exact value the user typed beats one you inferred.",
  "A value that matches a field's allowed set beats free text.",
  "When a term could belong to two fields, choose the one the surrounding words point at.",
];

const TIME_RULES = [
  "A lookback from now goes in since, written as a number and a unit: Nm for minutes, Nh for hours, Nd for days.",
  "Convert an unsupported unit to the nearest supported one, so a week becomes 7d.",
  "When a query names more than one lookback, use the longest.",
  "A range with a named start or end, such as a date, a clock time, or yesterday afternoon, goes in startTime and endTime as millisecond timestamps, not in since.",
  "Never answer with both since and startTime.",
];

function operatorsByField(spec: SearchSpec): string {
  return Object.entries(spec.config)
    .filter(([field]) => field in spec.fields)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      const plural = config.operators.length > 1 ? "s" : "";
      const values =
        "validValues" in config && config.validValues
          ? ` and must be one of: ${config.validValues.join(", ")}`
          : "";
      return `- ${field} (${spec.fields[field]}) accepts the ${operators} operator${plural}${values}`;
    })
    .join("\n");
}

function renderExample(example: Example): string {
  const note = example.note ? `\n// ${example.note}` : "";
  return `Query: ${JSON.stringify(example.query)}${note}\nResult: ${JSON.stringify(example.result, null, 2)}`;
}

export function buildSystemPrompt(spec: SearchSpec, referenceMs: number): string {
  const sections = [
    `You turn a plain-language question about ${spec.subject} into filters. Answer only with the filters the query asks for.`,
    `The current time is ${new Date(referenceMs).toISOString()}.`,
    `Fields:\n${operatorsByField(spec)}`,
    `Rules:\n${[...SHARED_RULES, ...(spec.rules ?? []), ...(spec.durationField ? TIME_RULES : [])]
      .map((rule, index) => `${index + 1}. ${rule}`)
      .join("\n")}`,
    `When two readings compete:\n${[...SHARED_PRIORITIES, ...(spec.priorities ?? [])]
      .map((rule, index) => `${index + 1}. ${rule}`)
      .join("\n")}`,
    `Examples:\n\n${spec.examples.map(renderExample).join("\n\n")}`,
  ];
  return sections.join("\n\n");
}
