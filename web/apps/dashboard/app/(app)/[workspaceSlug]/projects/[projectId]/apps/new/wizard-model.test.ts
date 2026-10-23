import { describe, expect, it } from "vitest";
import {
  type Resume,
  type WizardState,
  cardView,
  columnCards,
  flowGuard,
  initialWizardState,
  resolveCard,
  resumeWizard,
  stepProgress,
  visibleSteps,
  wizardReducer,
  wizardSearchParams,
} from "./wizard-model";

const fresh: Resume = { kind: "wizard", state: initialWizardState };

const resumedPane = (resume: Resume) =>
  resume.kind === "wizard" ? resolveCard(resume.state).pane : null;

const withApp = (patch: Partial<WizardState> = {}): WizardState => ({
  ...initialWizardState,
  source: "git",
  appId: "app_1",
  ...patch,
});

const image = (patch: Partial<WizardState> = {}): WizardState =>
  withApp({ source: "oci", ...patch });

describe("wizardReducer", () => {
  it("enters the source step when a source is picked", () => {
    const git = wizardReducer(initialWizardState, { type: "pick-source", source: "git" });
    const oci = wizardReducer(initialWizardState, { type: "pick-source", source: "oci" });
    expect(resolveCard(git).pane).toBe("pick-repo");
    expect(resolveCard(oci).pane).toBe("image");
  });

  it("walks the git steps to settings", () => {
    let state = withApp({ step: "source" });
    for (const expected of ["configure", "configure"]) {
      state = wizardReducer(state, { type: "next" });
      expect(resolveCard(state).pane).toBe(expected);
    }
  });

  it("walks the image steps to settings", () => {
    const state = wizardReducer(image({ step: "source" }), { type: "next" });
    expect(resolveCard(state).pane).toBe("configure");
    expect(resolveCard(wizardReducer(state, { type: "next" })).pane).toBe("configure");
  });

  it("returns the same state when going to the current step", () => {
    const state = withApp({ step: "settings" });
    expect(wizardReducer(state, { type: "go", step: "settings" })).toBe(state);
  });

  it("records the app once it exists", () => {
    expect(
      wizardReducer(initialWizardState, { type: "app-ready", appId: "app_1", source: "git" }),
    ).toEqual(withApp());
  });
});

describe("resolveCard", () => {
  it("asks for a source before anything else", () => {
    expect(resolveCard({ ...initialWizardState, step: "settings" })).toEqual({
      pane: "method",
      selected: null,
    });
  });

  it("falls back to the source step until the app exists", () => {
    const chosen = { ...initialWizardState, step: "settings" as const };
    expect(resolveCard({ ...chosen, source: "git" })).toEqual({ pane: "pick-repo", appId: null });
    expect(resolveCard({ ...chosen, source: "oci" })).toEqual({ pane: "image", appId: null });
  });

  it("skips the method step once the app has fixed its source", () => {
    expect(resolveCard(image({ step: "method" }))).toEqual({ pane: "image", appId: "app_1" });
  });

  it("gives settings the app and its source", () => {
    expect(resolveCard(image({ step: "settings" }))).toEqual({
      pane: "configure",
      appId: "app_1",
      source: "oci",
    });
    expect(resolveCard(withApp({ step: "settings" }))).toEqual({
      pane: "configure",
      appId: "app_1",
      source: "git",
    });
  });
});

describe("cardView", () => {
  it("steps back through the source's own steps, never into a source the app has fixed", () => {
    expect(cardView(initialWizardState, "method")?.previous).toBe(null);
    expect(cardView(withApp(), "pick-repo")?.previous).toBe(null);
    expect(cardView(withApp(), "configure")?.previous).toBe("source");
    expect(cardView({ ...initialWizardState, source: "oci" }, "image")?.previous).toBe("method");
  });

  it("labels a card by its place among the visible steps", () => {
    expect(cardView({ ...initialWizardState, source: "git" }, "pick-repo")?.label).toBe(
      "Step 2 of 3",
    );
    expect(cardView(withApp(), "configure")?.label).toBe("Step 2 of 2");
  });

  it("has no view for a card the state cannot reach", () => {
    expect(cardView(withApp(), "image")).toBe(null);
    expect(cardView(withApp(), "method")).toBe(null);
    expect(cardView({ ...initialWizardState, source: "git" }, "configure")).toBe(null);
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
  const at = (step: string | null) => ({ step, appId: "app_1" });

  it("starts fresh without an app", () => {
    expect(resumeWizard({ step: null, appId: null }, [app])).toEqual(fresh);
  });

  it("starts fresh when the app is not in this project", () => {
    expect(resumeWizard({ step: "configure", appId: "app_9" }, [app])).toEqual(fresh);
    expect(resumeWizard({ step: "pick-repo", appId: "app_9" }, [app])).toEqual(fresh);
  });

  it("resumes a chosen source on its source step before the app exists", () => {
    expect(resumeWizard({ step: "pick-repo", appId: null }, [])).toEqual({
      kind: "wizard",
      state: { ...initialWizardState, source: "git", step: "source" },
    });
    expect(resumeWizard({ step: "image", appId: null }, [])).toEqual({
      kind: "wizard",
      state: { ...initialWizardState, source: "oci", step: "source" },
    });
    expect(resumeWizard({ step: "configure", appId: null }, [])).toEqual(fresh);
  });

  it("resumes the exact card in the URL", () => {
    expect(resumedPane(resumeWizard(at("configure"), [app]))).toBe("configure");
    expect(resumedPane(resumeWizard(at("pick-repo"), [app]))).toBe("pick-repo");
  });

  it("sends a repository app without a connection to the picker", () => {
    const unlinked = [{ ...app, repositoryFullName: null }];
    expect(resumedPane(resumeWizard(at("configure"), unlinked))).toBe("pick-repo");
    expect(resumedPane(resumeWizard(at(null), unlinked))).toBe("pick-repo");
  });

  it("falls back to the configure card for an unknown step", () => {
    expect(resumedPane(resumeWizard(at("watch"), [app]))).toBe("configure");
    expect(resumedPane(resumeWizard(at("review"), [app]))).toBe("configure");
    expect(resumedPane(resumeWizard(at("constructor"), [app]))).toBe("configure");
    expect(resumedPane(resumeWizard(at(null), [app]))).toBe("configure");
  });

  it("resumes an image app on its own cards", () => {
    const oci = [{ ...app, sourceType: "oci" as const, repositoryFullName: null }];
    expect(resumeWizard(at("configure"), oci)).toEqual({
      kind: "wizard",
      state: { step: "settings", source: "oci", appId: "app_1" },
    });
    expect(resumedPane(resumeWizard(at("pick-repo"), oci))).toBe("configure");
  });

  it("sends an app that has deployed to its overview", () => {
    expect(resumeWizard(at("configure"), [{ ...app, currentDeploymentId: "d_1" }])).toEqual({
      kind: "app",
      appId: "app_1",
    });
    expect(resumeWizard(at("configure"), [{ ...app, headlineDeployment: { id: "d_1" } }])).toEqual({
      kind: "app",
      appId: "app_1",
    });
  });
});

describe("wizardSearchParams", () => {
  it("writes only the source's card before the app exists", () => {
    expect(wizardSearchParams({ ...initialWizardState, source: "git" })).toEqual({});
    expect(wizardSearchParams({ ...initialWizardState, source: "git", step: "source" })).toEqual({
      step: "pick-repo",
    });
    expect(wizardSearchParams({ ...initialWizardState, source: "oci", step: "settings" })).toEqual({
      step: "image",
    });
  });

  it("writes the app and the resolved card", () => {
    expect(wizardSearchParams(withApp({ step: "settings" }))).toEqual({
      appId: "app_1",
      step: "configure",
    });
    expect(wizardSearchParams(image({ step: "method" })).step).toBe("image");
  });
});

describe("visibleSteps", () => {
  it("lists the same three steps before the app exists", () => {
    expect(visibleSteps(initialWizardState)).toEqual([
      { id: "method", status: "current" },
      { id: "source", status: "todo" },
      { id: "settings", status: "todo" },
    ]);
    expect(
      visibleSteps({ ...initialWizardState, source: "oci", step: "source" }).map((s) => s.status),
    ).toEqual(["done", "current", "todo"]);
  });

  it("drops the method step once the app has fixed its source", () => {
    expect(visibleSteps(image({ step: "settings" }))).toEqual([
      { id: "source", status: "done" },
      { id: "settings", status: "current" },
    ]);
    expect(visibleSteps(withApp({ step: "source" })).map((s) => s.status)).toEqual([
      "current",
      "todo",
    ]);
  });
});

describe("stepProgress", () => {
  it("counts only the visible steps", () => {
    expect(stepProgress({ ...initialWizardState, source: "git", step: "source" })).toEqual({
      at: 1,
      total: 3,
      label: "Step 2 of 3",
    });
    expect(stepProgress(withApp({ step: "settings" }))).toEqual({
      at: 1,
      total: 2,
      label: "Step 2 of 2",
    });
  });
});

describe("columnCards", () => {
  it("peeks the repository picker before a source is chosen", () => {
    expect(columnCards(initialWizardState)).toEqual({
      at: 0,
      previous: null,
      current: "method",
      next: "pick-repo",
    });
  });

  it("peeks no method card once the app exists", () => {
    expect(columnCards(withApp({ step: "source" }))).toEqual({
      at: 0,
      previous: null,
      current: "pick-repo",
      next: "configure",
    });
    expect(columnCards(image({ step: "settings" })).previous).toBe("image");
  });
});

describe("flowGuard", () => {
  it("prompts before leaving an app that is still being set up", () => {
    expect(flowGuard({ hasApp: true, mutating: false, leaving: false })).toBe("confirm");
    expect(flowGuard({ hasApp: false, mutating: false, leaving: false })).toBe("off");
  });

  it("pins history while the app is being created", () => {
    expect(flowGuard({ hasApp: false, mutating: true, leaving: false })).toBe("hold");
  });

  it("stops guarding once deploy starts or the flow is leaving", () => {
    expect(flowGuard({ hasApp: true, mutating: true, leaving: false })).toBe("off");
    expect(flowGuard({ hasApp: true, mutating: false, leaving: true })).toBe("off");
    expect(flowGuard({ hasApp: false, mutating: true, leaving: true })).toBe("off");
  });
});
