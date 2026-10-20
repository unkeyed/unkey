import type { GuardMode } from "@/hooks/prevent-leave-model";
import {
  type App,
  type AppDeploymentRefs,
  latestDeploymentId,
} from "@/lib/collections/deploy/apps";

export type SourceKind = "git" | "oci";
export type Direction = "forward" | "back";

export type Step = "method" | "source" | "settings";
export type StepStatus = "done" | "current" | "todo";

const STEPS: readonly Step[] = ["method", "source", "settings"];

export type WizardState = {
  step: Step;
  source: SourceKind | null;
  appId: string | null;
};

export type Pane = "method" | "pick-repo" | "image" | "configure";

export type Card =
  | { pane: "method"; selected: SourceKind | null }
  | { pane: "pick-repo"; appId: string | null }
  | { pane: "image"; appId: string | null }
  | { pane: "configure"; appId: string; source: SourceKind };

const SOURCE_PANE: Record<SourceKind, "pick-repo" | "image"> = { git: "pick-repo", oci: "image" };

const PANE_STEP: Record<Pane, Step> = {
  method: "method",
  "pick-repo": "source",
  image: "source",
  configure: "settings",
};

const PEEK_SOURCE: SourceKind = "git";

export const sourceCopy: Record<SourceKind, { sourceLabel: string; hint: string }> = {
  git: {
    sourceLabel: "GitHub repository",
    hint: "Build from source and deploy each push",
  },
  oci: {
    sourceLabel: "Container image",
    hint: "Run a prebuilt image from a public registry",
  },
};

export const cardCopy: Record<Pane, { title: string; description: string; pending: string }> = {
  method: {
    title: "Deploy a new app",
    description: "Choose where your app's code comes from.",
    pending: "Loading…",
  },
  "pick-repo": {
    title: "Import a GitHub repository",
    description: "Choose a repository to deploy.",
    pending: "Loading your repositories…",
  },
  image: {
    title: "Run a container image",
    description: "Pin a tag or digest so every deploy runs the same image.",
    pending: "Loading your image…",
  },
  configure: {
    title: "Set up your app",
    description: "Check the settings, then deploy. You can change them at any time.",
    pending: "Loading your settings…",
  },
};

export const footerlessPanes: ReadonlySet<Pane> = new Set(["method"]);

export function resolveCard({ step, source, appId }: WizardState): Card {
  if (source === null || (step === "method" && appId === null)) {
    return { pane: "method", selected: source };
  }
  if (step === "settings" && appId !== null) {
    return { pane: "configure", appId, source };
  }
  return { pane: SOURCE_PANE[source], appId };
}

function visibleStepIds(state: WizardState): readonly Step[] {
  return state.appId === null ? STEPS : STEPS.filter((step) => step !== "method");
}

function currentStep(state: WizardState): Step {
  return PANE_STEP[resolveCard(state).pane];
}

function paneOf(step: Step, source: SourceKind | null): Pane {
  const panes: Record<Step, Pane> = {
    method: "method",
    source: SOURCE_PANE[source ?? PEEK_SOURCE],
    settings: "configure",
  };
  return panes[step];
}

export function columnCards(state: WizardState) {
  const steps = visibleStepIds(state);
  const at = steps.indexOf(currentStep(state));
  const pane = (step: Step | undefined) => (step ? paneOf(step, state.source) : null);
  return {
    at,
    previous: pane(steps[at - 1]),
    current: paneOf(steps[at], state.source),
    next: pane(steps[at + 1]),
  };
}

export type CardView = { card: Card; previous: Step | null; label: string };

export function cardView(state: WizardState, pane: Pane): CardView | null {
  const at = { ...state, step: PANE_STEP[pane] };
  const card = resolveCard(at);
  if (card.pane !== pane) {
    return null;
  }
  const steps = visibleStepIds(at);
  const index = steps.indexOf(at.step);
  return {
    card,
    previous: steps[index - 1] ?? null,
    label: `Step ${index + 1} of ${steps.length}`,
  };
}

export const initialWizardState: WizardState = {
  step: "method",
  source: null,
  appId: null,
};

export type WizardAction =
  | { type: "pick-source"; source: SourceKind }
  | { type: "go"; step: Step }
  | { type: "next" }
  | { type: "app-ready"; appId: string; source: SourceKind };

function goTo(state: WizardState, step: Step): WizardState {
  return state.step === step ? state : { ...state, step };
}

export function wizardReducer(state: WizardState, action: WizardAction): WizardState {
  switch (action.type) {
    case "pick-source":
      return { ...state, source: action.source, step: "source" };
    case "go":
      return goTo(state, action.step);
    case "next": {
      const steps = visibleStepIds(state);
      return goTo(state, steps[steps.indexOf(currentStep(state)) + 1] ?? state.step);
    }
    case "app-ready":
      return { ...state, source: action.source, appId: action.appId };
  }
}

type ResumeParams = { step: string | null; appId: string | null };

type ResumableApp = Pick<App, "id" | "sourceType" | "repositoryFullName"> & AppDeploymentRefs;

function isPane(value: string): value is Pane {
  return Object.hasOwn(PANE_STEP, value);
}

export function pendingPane(step: string | null): Pane {
  return step !== null && isPane(step) ? step : "method";
}

function resumeStep(token: string | null, app: ResumableApp, source: SourceKind): Step {
  const own = token !== null && isPane(token) && paneOf(PANE_STEP[token], source) === token;
  const step = own ? PANE_STEP[token] : "settings";
  if (source === "git" && app.repositoryFullName === null && step === "settings") {
    return "source";
  }
  return step;
}

export type Resume = { kind: "wizard"; state: WizardState } | { kind: "app"; appId: string };

const startFresh: Resume = { kind: "wizard", state: initialWizardState };

function chosenSource(token: string | null): SourceKind | null {
  const sources: SourceKind[] = ["git", "oci"];
  return sources.find((source) => SOURCE_PANE[source] === token) ?? null;
}

export function resumeWizard(params: ResumeParams, apps: readonly ResumableApp[]): Resume {
  const app = params.appId ? apps.find((a) => a.id === params.appId) : undefined;
  if (!app) {
    const chosen = params.appId ? null : chosenSource(params.step);
    return chosen
      ? {
          kind: "wizard",
          state: wizardReducer(initialWizardState, { type: "pick-source", source: chosen }),
        }
      : startFresh;
  }
  if (latestDeploymentId(app) !== null) {
    return { kind: "app", appId: app.id };
  }
  const source = app.sourceType === "oci" ? "oci" : "git";
  return {
    kind: "wizard",
    state: { step: resumeStep(params.step, app, source), source, appId: app.id },
  };
}

export function wizardSearchParams(state: WizardState): Record<string, string> {
  const { pane } = resolveCard(state);
  if (state.appId === null) {
    return pane === "method" ? {} : { step: pane };
  }
  return { appId: state.appId, step: pane };
}

export type StepRow = { id: Step; status: StepStatus };

export const stepTitle: Record<Step, string> = {
  method: "Select a source",
  source: "Choose your app",
  settings: "Configure and deploy",
};

function stepStatus(index: number, at: number): StepStatus {
  if (index < at) {
    return "done";
  }
  return index === at ? "current" : "todo";
}

export function visibleSteps(state: WizardState): StepRow[] {
  const steps = visibleStepIds(state);
  const at = steps.indexOf(currentStep(state));
  return steps.map((id, index) => ({ id, status: stepStatus(index, at) }));
}

export function stepProgress(state: WizardState): { at: number; total: number; label: string } {
  const steps = visibleSteps(state);
  const at = steps.findIndex((step) => step.status === "current");
  return { at, total: steps.length, label: `Step ${at + 1} of ${steps.length}` };
}

export function flowGuard({
  hasApp,
  mutating,
  leaving,
}: {
  hasApp: boolean;
  mutating: boolean;
  leaving: boolean;
}): GuardMode {
  if (leaving) {
    return "off";
  }
  if (mutating) {
    return hasApp ? "off" : "hold";
  }
  return hasApp ? "confirm" : "off";
}
