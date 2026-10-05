import { describe, expect, it } from "vitest";
import {
  type WizardState,
  flowLocked,
  initialWizardState,
  previousCard,
  resolveCard,
  resumeFromApps,
  resumeWizard,
  wizardReducer,
  wizardSearchParams,
} from "./wizard-model";

const withApp = (patch: Partial<WizardState> = {}): WizardState => ({
  ...initialWizardState,
  source: "git",
  app: { id: "app_1", source: "git" },
  ...patch,
});

const image = (patch: Partial<WizardState> = {}): WizardState =>
  withApp({ source: "oci", app: { id: "app_1", source: "oci" }, ...patch });

describe("wizardReducer", () => {
  it("enters the source's first card when a source is picked", () => {
    expect(wizardReducer(initialWizardState, { type: "pick-source", source: "git" }).card).toBe(
      "pick-repo",
    );
    expect(wizardReducer(initialWizardState, { type: "pick-source", source: "oci" }).card).toBe(
      "image",
    );
  });

  it("walks the git cards past variables", () => {
    const steps = ["pick-repo", "configure-repo", "review"] as const;
    let state = withApp({ card: "pick-repo" });
    for (const expected of steps.slice(1)) {
      state = wizardReducer(state, { type: "next" });
      expect(state.card).toBe(expected);
    }
  });

  it("walks the image cards through variables", () => {
    const state = wizardReducer(image({ card: "image" }), { type: "next" });
    expect(state.card).toBe("variables");
    expect(wizardReducer(state, { type: "next" }).card).toBe("review");
  });

  it("returns the same state when going to the current card", () => {
    const state = withApp({ card: "review" });
    expect(wizardReducer(state, { type: "go", card: "review" })).toBe(state);
  });

  it("clears the field focus on any move", () => {
    const state = withApp({ card: "configure-repo", focus: "dockerfile" });
    expect(wizardReducer(state, { type: "next" })).toEqual(withApp({ card: "review" }));
  });

  it("watches a new deployment", () => {
    expect(
      wizardReducer(withApp({ card: "review" }), {
        type: "deployment-created",
        deploymentId: "d_1",
      }),
    ).toEqual(withApp({ card: "watch", deploymentId: "d_1" }));
  });

  it("edits settings on the source's configure card with a focus", () => {
    const deployed = withApp({ card: "watch", deploymentId: "d_1" });
    expect(wizardReducer(deployed, { type: "edit-settings", focus: "dockerfile" })).toEqual(
      withApp({ card: "configure-repo", focus: "dockerfile" }),
    );
    expect(
      wizardReducer(image({ card: "watch", deploymentId: "d_1" }), {
        type: "edit-settings",
        focus: null,
      }).card,
    ).toBe("image");
  });

  it("tracks app creation", () => {
    const pending = wizardReducer(initialWizardState, { type: "create-start" });
    expect(pending.pending).toBe(true);
    expect(wizardReducer(pending, { type: "create-failed" }).pending).toBe(false);
    expect(wizardReducer(pending, { type: "app-ready", appId: "app_1", source: "git" })).toEqual(
      withApp(),
    );
  });
});

describe("flowLocked", () => {
  it("locks while an app is being created and once a deploy starts", () => {
    expect(flowLocked(initialWizardState)).toBe(false);
    expect(flowLocked({ ...initialWizardState, pending: true })).toBe(true);
    expect(flowLocked(withApp({ deploymentId: "d_1" }))).toBe(true);
  });
});

describe("resolveCard", () => {
  it("asks for a source before anything else", () => {
    expect(resolveCard({ ...initialWizardState, card: "variables" })).toEqual({
      id: "source",
      selected: null,
      lockedTo: null,
    });
  });

  it("falls back to the entry card until the app exists", () => {
    const chosen = { ...initialWizardState, card: "review" as const };
    expect(resolveCard({ ...chosen, source: "git" })).toEqual({ id: "pick-repo", appId: null });
    expect(resolveCard({ ...chosen, source: "oci" })).toEqual({ id: "image", appId: null });
  });

  it("falls back to the entry card for a card of the other source", () => {
    expect(resolveCard(withApp({ card: "variables" }))).toEqual({
      id: "pick-repo",
      appId: "app_1",
    });
  });

  it("takes the source from the app and locks it", () => {
    expect(resolveCard(withApp({ card: "source", source: "oci" }))).toEqual({
      id: "source",
      selected: "git",
      lockedTo: "git",
    });
  });

  it("carries the field focus on the configure card", () => {
    expect(resolveCard(withApp({ card: "configure-repo", focus: "dockerfile" }))).toEqual({
      id: "configure-repo",
      appId: "app_1",
      focus: "dockerfile",
    });
  });

  it("shows watch and result only with a deployment", () => {
    expect(resolveCard(withApp({ card: "watch" }))).toEqual({
      id: "review",
      appId: "app_1",
      source: "git",
    });
    expect(resolveCard(withApp({ card: "result", deploymentId: "d_1" }))).toEqual({
      id: "result",
      appId: "app_1",
      source: "git",
      deploymentId: "d_1",
    });
  });
});

describe("previousCard", () => {
  it("steps back through the source's own cards", () => {
    expect(previousCard(initialWizardState)).toBe(null);
    expect(previousCard(withApp({ card: "pick-repo" }))).toBe("source");
    expect(previousCard(withApp({ card: "review" }))).toBe("configure-repo");
    expect(previousCard(image({ card: "review" }))).toBe("variables");
  });
});

describe("resumeWizard", () => {
  const app = {
    id: "app_1",
    sourceType: "git" as const,
    repositoryConnected: true,
    latestDeploymentId: null,
  };
  const at = (step: string | null, deploymentId: string | null = null) => ({
    step,
    appId: "app_1",
    deploymentId,
  });

  it("starts fresh without an app", () => {
    expect(resumeWizard({ step: null, appId: null, deploymentId: null }, null)).toBe(
      initialWizardState,
    );
  });

  it("starts fresh when the app is not in this project", () => {
    expect(resumeWizard({ step: "variables", appId: "app_9", deploymentId: null }, null)).toBe(
      initialWizardState,
    );
  });

  it("resumes the GitHub install return on the repository picker", () => {
    expect(resumeWizard(at("select-repo"), app)).toEqual(
      withApp({ source: null, card: "pick-repo" }),
    );
  });

  it("resumes the exact card in the URL", () => {
    expect(resumeWizard(at("configure-repo"), app).card).toBe("configure-repo");
    expect(resumeWizard(at("pick-repo"), app).card).toBe("pick-repo");
    expect(resumeWizard(at("source"), app).card).toBe("source");
  });

  it("sends a repository app without a connection to the picker", () => {
    const unlinked = { ...app, repositoryConnected: false };
    expect(resumeWizard(at("configure-repo"), unlinked).card).toBe("pick-repo");
    expect(resumeWizard(at("configure"), unlinked).card).toBe("pick-repo");
  });

  it("maps the old step names", () => {
    expect(resumeWizard(at("configure"), app).card).toBe("configure-repo");
    expect(resumeWizard(at("deploy"), app).card).toBe("review");
    expect(resumeWizard(at(null), app).card).toBe("configure-repo");
  });

  it("resumes an image app on its own cards", () => {
    const oci = { ...app, sourceType: "oci" as const, repositoryConnected: false };
    expect(resumeWizard(at("variables"), oci)).toEqual({
      ...initialWizardState,
      card: "variables",
      app: { id: "app_1", source: "oci" },
    });
    expect(resumeWizard(at("configure-repo"), oci).card).toBe("image");
    expect(resumeWizard(at("configure"), oci).card).toBe("image");
  });

  it("keeps an undeployed app off the deployment cards", () => {
    expect(resumeWizard(at("watch"), app).card).toBe("review");
  });

  it("resumes its own deployment on watch or result", () => {
    const deployed = { ...app, latestDeploymentId: "d_1" };
    expect(resumeWizard(at("deploy", "d_1"), deployed)).toEqual(
      withApp({ source: null, card: "watch", deploymentId: "d_1" }),
    );
    expect(resumeWizard(at("result", "d_1"), deployed).card).toBe("result");
  });

  it("refuses an app that already deployed outside this flow", () => {
    const deployed = { ...app, latestDeploymentId: "d_1" };
    expect(resumeWizard(at("review"), deployed)).toBe(initialWizardState);
  });
});

describe("resumeFromApps", () => {
  const params = { step: "watch", appId: "app_1", deploymentId: "d_old" };
  const project = (latest: string | null) => [
    {
      id: "app_1",
      sourceType: "git" as const,
      repositoryFullName: "acme/api",
      headlineDeployment: latest ? { id: latest } : null,
      currentDeploymentId: latest,
    },
  ];

  it("resumes the deployment only when it is the app's latest", () => {
    expect(resumeFromApps({ ...params, deploymentId: "d_new" }, project("d_new"))).toEqual(
      withApp({ source: null, card: "watch", deploymentId: "d_new" }),
    );
  });

  it("refuses a stale deployment id from the URL", () => {
    expect(resumeFromApps(params, project("d_new"))).toBe(initialWizardState);
  });
});

describe("wizardSearchParams", () => {
  it("writes nothing before the app exists", () => {
    expect(wizardSearchParams({ ...initialWizardState, source: "git" })).toEqual({});
  });

  it("writes the app, the resolved card and the deployment", () => {
    expect(wizardSearchParams(withApp({ card: "result", deploymentId: "d_1" }))).toEqual({
      appId: "app_1",
      step: "result",
      deploymentId: "d_1",
    });
    expect(wizardSearchParams(withApp({ card: "variables" })).step).toBe("pick-repo");
  });
});
