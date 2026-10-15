import type { SourceKind } from "../../wizard-model";

export type SourceAction = "pick" | "connect-github";

export function sourceAction(kind: SourceKind, needsGithub: boolean): SourceAction {
  return kind === "git" && needsGithub ? "connect-github" : "pick";
}
