import { type SourceKind, sourceCopy } from "../../wizard-model";

type Size = { cpuMillicores: number; memoryMib: number };

export type ReviewValue =
  | { type: "loading" }
  | { type: "text"; text: string }
  | { type: "mono"; text: string }
  | { type: "size"; size: Size }
  | { type: "regions"; names: string[] };

export type ReviewRow = { label: string; value: ReviewValue };

type ReviewInput = {
  source: SourceKind;
  app: { name: string; imageReference: string | null } | undefined;
  repository: { fullName: string; branch: string } | null | undefined;
  runtime: { port: number; regions: string[]; size: Size } | undefined;
  variableCount: number;
};

const loading: ReviewValue = { type: "loading" };
const text = (value: string): ReviewValue => ({ type: "text", text: value });
const mono = (value: string): ReviewValue => ({ type: "mono", text: value });

function origin(input: ReviewInput): ReviewValue {
  if (input.source === "oci") {
    if (!input.app) {
      return loading;
    }
    return input.app.imageReference ? mono(input.app.imageReference) : text("None");
  }
  if (input.repository === undefined) {
    return loading;
  }
  return input.repository
    ? mono(`${input.repository.fullName} · ${input.repository.branch}`)
    : text("None");
}

function regions(names: string[]): ReviewValue {
  return names.length > 0 ? { type: "regions", names } : text("None");
}

export function reviewRows(input: ReviewInput): ReviewRow[] {
  const { runtime } = input;
  return [
    { label: "Source", value: text(sourceCopy[input.source].sourceLabel) },
    { label: sourceCopy[input.source].originLabel, value: origin(input) },
    {
      label: "App name",
      value: input.app ? text(input.app.name) : loading,
    },
    { label: "Port", value: runtime ? mono(String(runtime.port)) : loading },
    {
      label: "Regions",
      value: runtime ? regions(runtime.regions) : loading,
    },
    { label: "Size", value: runtime ? { type: "size", size: runtime.size } : loading },
    {
      label: "Environment variables",
      value: text(input.variableCount === 0 ? "None" : `${input.variableCount} set`),
    },
  ];
}
