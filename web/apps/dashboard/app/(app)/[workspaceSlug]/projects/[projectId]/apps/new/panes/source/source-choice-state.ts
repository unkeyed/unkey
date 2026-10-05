import { type SourceKind, sourceCopy } from "../../wizard-model";

type SourceAction = "pick" | "connect-github" | "locked";

export type SourceChoice = {
  kind: SourceKind;
  title: string;
  description: string;
  action: SourceAction;
};

type SourceChoiceInput = { lockedTo: SourceKind | null; needsGithub: boolean };

const SOURCE_KINDS: readonly SourceKind[] = ["git", "oci"];

function sourceAction(
  kind: SourceKind,
  { lockedTo, needsGithub }: SourceChoiceInput,
): SourceAction {
  if (lockedTo !== null && lockedTo !== kind) {
    return "locked";
  }
  return kind === "git" && needsGithub ? "connect-github" : "pick";
}

const lockedDescription: Record<SourceKind, string> = {
  git: "This app uses a container image. Create a new app to use a repository.",
  oci: "This app uses a GitHub repository. Create a new app to use an image.",
};

const actionCopy: Record<
  SourceAction,
  (kind: SourceKind) => { title: string; description: string }
> = {
  pick: (kind) => ({ title: sourceCopy[kind].sourceLabel, description: sourceCopy[kind].hint }),
  "connect-github": () => ({
    title: "Connect GitHub",
    description: "Install the Unkey GitHub app to import a repository",
  }),
  locked: (kind) => ({
    title: sourceCopy[kind].sourceLabel,
    description: lockedDescription[kind],
  }),
};

export function sourceChoices(input: SourceChoiceInput): SourceChoice[] {
  return SOURCE_KINDS.map((kind) => {
    const action = sourceAction(kind, input);
    return { kind, action, ...actionCopy[action](kind) };
  });
}
