import type { SourceKind } from "../../wizard-model";

export type SourceAction = "pick" | "connect-github" | "locked";

export function sourceAction(
  kind: SourceKind,
  { lockedTo, needsGithub }: { lockedTo: SourceKind | null; needsGithub: boolean },
): SourceAction {
  if (lockedTo !== null && lockedTo !== kind) {
    return "locked";
  }
  return kind === "git" && needsGithub ? "connect-github" : "pick";
}
