import { describe, expect, it } from "vitest";
import {
  type Resume,
  type WizardState,
  flowLocked,
  initialWizardState,
  previousCard,
  resolveCard,
  resumeWizard,
  wizardReducer,
  wizardSearchParams,
} from "./wizard-model";

const fresh: Resume = { kind: "wizard", state: initialWizardState };

const resumedCard = (resume: Resume) => (resume.kind === "wizard" ? resume.state.card : null);

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
    repositoryFullName: "acme/api",
    headlineDeployment: null,
    currentDeploymentId: null,
  };
  const deployed = (id: string) => ({
    ...app,
    headlineDeployment: { id },
    currentDeploymentId: id,
  });
  const at = (step: string | null, deploymentId: string | null = null) => ({
    step,
    appId: "app_1",
    deploymentId,
  });

  it("starts fresh without an app", () => {
    expect(resumeWizard({ step: null, appId: null, deploymentId: null }, [app])).toEqual(fresh);
  });

  it("starts fresh when the app is not in this project", () => {
    expect(resumeWizard({ step: "variables", appId: "app_9", deploymentId: null }, [app])).toEqual(
      fresh,
    );
  });

  it("resumes the GitHub install return on the repository picker", () => {
    expect(resumeWizard(at("select-repo"), [app])).toEqual({
      kind: "wizard",
      state: withApp({ source: null, card: "pick-repo" }),
    });
  });

  it("resumes the exact card in the URL", () => {
    expect(resumedCard(resumeWizard(at("configure-repo"), [app]))).toBe("configure-repo");
    expect(resumedCard(resumeWizard(at("pick-repo"), [app]))).toBe("pick-repo");
    expect(resumedCard(resumeWizard(at("source"), [app]))).toBe("source");
  });

  it("sends a repository app without a connection to the picker", () => {
    const unlinked = [{ ...app, repositoryFullName: null }];
    expect(resumedCard(resumeWizard(at("configure-repo"), unlinked))).toBe("pick-repo");
    expect(resumedCard(resumeWizard(at("configure"), unlinked))).toBe("pick-repo");
  });

  it("falls back to the configure card for an unknown step", () => {
    expect(resumedCard(resumeWizard(at("configure"), [app]))).toBe("configure-repo");
    expect(resumedCard(resumeWizard(at(null), [app]))).toBe("configure-repo");
  });

  it("resumes an image app on its own cards", () => {
    const oci = [{ ...app, sourceType: "oci" as const, repositoryFullName: null }];
    expect(resumeWizard(at("variables"), oci)).toEqual({
      kind: "wizard",
      state: { ...initialWizardState, card: "variables", app: { id: "app_1", source: "oci" } },
    });
    expect(resumedCard(resumeWizard(at("configure-repo"), oci))).toBe("image");
    expect(resumedCard(resumeWizard(at("configure"), oci))).toBe("image");
  });

  it("keeps an undeployed app off the deployment cards", () => {
    expect(resumedCard(resumeWizard(at("watch"), [app]))).toBe("review");
  });

  it("resumes its own deployment on watch or result", () => {
    expect(resumeWizard(at("watch", "d_1"), [deployed("d_1")])).toEqual({
      kind: "wizard",
      state: withApp({ source: null, card: "watch", deploymentId: "d_1" }),
    });
    expect(resumedCard(resumeWizard(at("result", "d_1"), [deployed("d_1")]))).toBe("result");
  });

  it("sends an app that already deployed outside this flow to its overview", () => {
    expect(resumeWizard(at("review"), [deployed("d_1")])).toEqual({ kind: "app", appId: "app_1" });
    expect(resumeWizard(at("watch", "d_0"), [deployed("d_1")])).toEqual({
      kind: "app",
      appId: "app_1",
    });
  });

  it("reads the latest deployment from the headline before the current one", () => {
    const moved = [{ ...app, headlineDeployment: { id: "d_new" }, currentDeploymentId: "d_old" }];
    expect(resumeWizard(at("watch", "d_old"), moved)).toEqual({ kind: "app", appId: "app_1" });
    expect(resumedCard(resumeWizard(at("watch", "d_new"), moved))).toBe("watch");
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
