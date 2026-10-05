export type SourceKind = "git" | "oci";
export type Direction = "forward" | "back";
export type SetupFieldFocus = "dockerfile";

export type CardId =
  | "source"
  | "pick-repo"
  | "configure-repo"
  | "image"
  | "variables"
  | "review"
  | "watch"
  | "result";

export const sourceCards: Record<SourceKind, readonly CardId[]> = {
  git: ["source", "pick-repo", "configure-repo", "review", "watch", "result"],
  oci: ["source", "image", "variables", "review", "watch", "result"],
};

const unchosenCards: readonly CardId[] = ["source"];

const configureCards: Record<SourceKind, CardId> = { git: "configure-repo", oci: "image" };

type WizardApp = { id: string; source: SourceKind };

export type WizardState = {
  card: CardId;
  source: SourceKind | null;
  app: WizardApp | null;
  deploymentId: string | null;
  focus: SetupFieldFocus | null;
  pending: boolean;
};

export type Card =
  | { id: "source"; selected: SourceKind | null; lockedTo: SourceKind | null }
  | { id: "pick-repo"; appId: string | null }
  | { id: "configure-repo"; appId: string; focus: SetupFieldFocus | null }
  | { id: "image"; appId: string | null }
  | { id: "variables"; appId: string }
  | { id: "review"; appId: string; source: SourceKind }
  | { id: "watch"; appId: string; source: SourceKind; deploymentId: string }
  | { id: "result"; appId: string; source: SourceKind; deploymentId: string };

export type ConfigCard = Exclude<Card, { id: "watch" | "result" }>;
export type ConfigCardId = ConfigCard["id"];

export function isConfigCard(card: Card): card is ConfigCard {
  return card.id !== "watch" && card.id !== "result";
}

export const sourceCopy: Record<
  SourceKind,
  { sourceLabel: string; originLabel: string; hint: string }
> = {
  git: {
    sourceLabel: "GitHub repository",
    originLabel: "Repository",
    hint: "Build from source and deploy each push",
  },
  oci: {
    sourceLabel: "Container image",
    originLabel: "Image",
    hint: "Run a prebuilt image from a public registry",
  },
};

type CardCopy = { title: string; description: string };

export const cardCopy: Record<ConfigCardId, CardCopy> = {
  source: {
    title: "Deploy a new app",
    description: "Choose where your app's code comes from.",
  },
  "pick-repo": {
    title: "Import a GitHub repository",
    description: "Choose a repository to deploy.",
  },
  "configure-repo": {
    title: "Set up your app",
    description: "Check the detected settings, then continue.",
  },
  image: {
    title: "Run a container image",
    description: "Pin a tag or digest so every deploy runs the same image.",
  },
  variables: {
    title: "Environment variables",
    description: "Values your app reads at runtime. You can change them at any time.",
  },
  review: {
    title: "Review and deploy",
    description: "Your first deployment uses these settings.",
  },
};

export const cardHasFooter: Record<ConfigCardId, boolean> = {
  source: false,
  "pick-repo": true,
  "configure-repo": true,
  image: true,
  variables: true,
  review: true,
};

export function wizardSource(state: WizardState): SourceKind | null {
  return state.app?.source ?? state.source;
}

export function cardList(state: WizardState): readonly CardId[] {
  const source = wizardSource(state);
  return source ? sourceCards[source] : unchosenCards;
}

function entryCard(source: SourceKind, appId: string | null): Card {
  return { id: source === "git" ? "pick-repo" : "image", appId };
}

// A card that its data cannot back yet falls back to the nearest earlier card
// that it can, so a stale URL or a neighbour lookup never renders a broken pane.
export function resolveCard(state: WizardState): Card {
  const source = wizardSource(state);
  const { app } = state;
  if (state.card === "source" || source === null) {
    return { id: "source", selected: source, lockedTo: app?.source ?? null };
  }
  if (
    !sourceCards[source].includes(state.card) ||
    state.card === "pick-repo" ||
    state.card === "image" ||
    !app
  ) {
    return entryCard(source, app?.id ?? null);
  }
  if (state.card === "configure-repo") {
    return { id: "configure-repo", appId: app.id, focus: state.focus };
  }
  if (state.card === "variables") {
    return { id: "variables", appId: app.id };
  }
  if (state.card === "review" || state.deploymentId === null) {
    return { id: "review", appId: app.id, source: app.source };
  }
  return { id: state.card, appId: app.id, source: app.source, deploymentId: state.deploymentId };
}

export function previousCard(state: WizardState): CardId | null {
  const cards = cardList(state);
  return cards[cards.indexOf(resolveCard(state).id) - 1] ?? null;
}

export function flowLocked(state: WizardState): boolean {
  return state.pending || state.deploymentId !== null;
}

export const initialWizardState: WizardState = {
  card: "source",
  source: null,
  app: null,
  deploymentId: null,
  focus: null,
  pending: false,
};

export type WizardAction =
  | { type: "pick-source"; source: SourceKind }
  | { type: "go"; card: CardId }
  | { type: "next" }
  | { type: "create-start" }
  | { type: "create-failed" }
  | { type: "app-ready"; appId: string; source: SourceKind }
  | { type: "deployment-created"; deploymentId: string }
  | { type: "edit-settings"; focus: SetupFieldFocus | null };

function goTo(state: WizardState, card: CardId): WizardState {
  if (state.card === card && state.focus === null) {
    return state;
  }
  return { ...state, card, focus: null };
}

export function wizardReducer(state: WizardState, action: WizardAction): WizardState {
  switch (action.type) {
    case "pick-source":
      return {
        ...state,
        source: action.source,
        card: entryCard(action.source, null).id,
        focus: null,
      };
    case "go":
      return goTo(state, action.card);
    case "next": {
      const cards = cardList(state);
      const at = cards.indexOf(resolveCard(state).id);
      return goTo(state, cards[at + 1] ?? state.card);
    }
    case "create-start":
      return { ...state, pending: true };
    case "create-failed":
      return { ...state, pending: false };
    case "app-ready":
      return {
        ...state,
        source: action.source,
        app: { id: action.appId, source: action.source },
        pending: false,
      };
    case "deployment-created":
      return { ...state, deploymentId: action.deploymentId, card: "watch", focus: null };
    case "edit-settings": {
      const source = wizardSource(state);
      if (source === null) {
        return state;
      }
      return {
        ...state,
        deploymentId: null,
        card: configureCards[source],
        focus: action.focus,
      };
    }
  }
}

// routes.projects.apps.new builds this step only for the GitHub install callback.
export const GITHUB_RETURN_STEP = "select-repo";

type ResumeParams = {
  step: string | null;
  appId: string | null;
  deploymentId: string | null;
};

type ResumableApp = {
  id: string;
  sourceType: "git" | "oci" | "unknown";
  repositoryConnected: boolean;
  latestDeploymentId: string | null;
};

const cardIds: readonly string[] = [...new Set(Object.values(sourceCards).flat())];

function isCardId(step: string): step is CardId {
  return cardIds.includes(step);
}

function resumeCard(step: string | null, app: ResumableApp, source: SourceKind): CardId {
  if (step === GITHUB_RETURN_STEP) {
    return "pick-repo";
  }
  if (step === "deploy") {
    return "review";
  }
  const known = step !== null && isCardId(step) && sourceCards[source].includes(step);
  const card = known ? step : configureCards[source];
  if (card === "configure-repo" && !app.repositoryConnected) {
    return "pick-repo";
  }
  return card;
}

// Only an app this flow could have made resumes: one that has not deployed
// yet, or whose latest deployment is the one the URL names.
export function resumeWizard(params: ResumeParams, app: ResumableApp | null): WizardState {
  if (!params.appId || !app || app.id !== params.appId) {
    return initialWizardState;
  }
  const source = app.sourceType === "oci" ? "oci" : "git";
  const card = resumeCard(params.step, app, source);
  const resumed: WizardState = { ...initialWizardState, app: { id: app.id, source }, card };
  if (app.latestDeploymentId === null) {
    return card === "watch" || card === "result" ? { ...resumed, card: "review" } : resumed;
  }
  if (params.deploymentId === app.latestDeploymentId) {
    return {
      ...resumed,
      card: card === "result" ? "result" : "watch",
      deploymentId: app.latestDeploymentId,
    };
  }
  return initialWizardState;
}

type ProjectApp = {
  id: string;
  sourceType: ResumableApp["sourceType"];
  repositoryFullName: string | null;
  headlineDeployment: { id: string } | null;
  currentDeploymentId: string | null;
};

export function resumeFromApps(params: ResumeParams, apps: readonly ProjectApp[]): WizardState {
  const app = apps.find((a) => a.id === params.appId);
  return resumeWizard(
    params,
    app
      ? {
          id: app.id,
          sourceType: app.sourceType,
          repositoryConnected: app.repositoryFullName !== null,
          latestDeploymentId: app.headlineDeployment?.id ?? app.currentDeploymentId,
        }
      : null,
  );
}

export function wizardSearchParams(state: WizardState): Record<string, string> {
  if (!state.app) {
    return {};
  }
  const params: Record<string, string> = {
    appId: state.app.id,
    step: resolveCard(state).id,
  };
  if (state.deploymentId) {
    params.deploymentId = state.deploymentId;
  }
  return params;
}
