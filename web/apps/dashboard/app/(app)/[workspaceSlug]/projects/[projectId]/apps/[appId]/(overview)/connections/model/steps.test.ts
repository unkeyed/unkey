import { describe, expect, it } from "vitest";
import { FORM_MODES } from "./mode";
import {
  FORM_STEPS,
  type StepsState,
  completeStep,
  firstStepWithError,
  openStep,
  pickedApp,
  stepStatus,
} from "./steps";

const statuses = (state: StepsState) => FORM_STEPS.map((step) => stepStatus(state, step));

describe("form steps", () => {
  it("walks add from app to config to code", () => {
    const add = FORM_MODES.add.initialSteps;
    expect(statuses(add)).toEqual(["open", "todo", "todo"]);

    const configuring = pickedApp(add, true);
    expect(statuses(configuring)).toEqual(["done", "open", "todo"]);
    expect(statuses(completeStep(configuring))).toEqual(["done", "done", "open"]);
  });

  it("opens edit on the code step", () => {
    expect(statuses(FORM_MODES.edit.initialSteps)).toEqual(["done", "done", "open"]);
  });

  it("returns to the code step after config is reopened and continued", () => {
    const reopened = openStep(FORM_MODES.edit.initialSteps, "config");
    expect(statuses(reopened)).toEqual(["done", "open", "todo"]);
    expect(statuses(completeStep(reopened))).toEqual(["done", "done", "open"]);
  });

  it("skips config when the same app is picked again", () => {
    const done = completeStep(pickedApp(FORM_MODES.add.initialSteps, true));
    expect(statuses(pickedApp(openStep(done, "app"), false))).toEqual(["done", "done", "open"]);
  });

  it("reopens config when a different app is picked", () => {
    const done = completeStep(pickedApp(FORM_MODES.add.initialSteps, true));
    expect(statuses(pickedApp(openStep(done, "app"), true))).toEqual(["done", "open", "todo"]);
  });

  it("leaves a state with no open step alone", () => {
    const closed: StepsState = { open: null, done: new Set(FORM_STEPS) };
    expect(completeStep(closed)).toBe(closed);
  });
});

describe("firstStepWithError", () => {
  it("finds the first step that holds an invalid field", () => {
    expect(firstStepWithError({})).toBeNull();
    expect(firstStepWithError({ target: {} })).toBe("config");
    expect(firstStepWithError({ name: {}, targetAppId: {} })).toBe("app");
  });
});
