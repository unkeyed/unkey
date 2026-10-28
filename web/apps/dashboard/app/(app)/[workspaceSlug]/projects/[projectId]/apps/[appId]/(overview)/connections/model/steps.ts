export const FORM_STEPS = ["app", "config"] as const;

export type FormStep = (typeof FORM_STEPS)[number];

export type StepStatus = "open" | "done" | "todo";

export type StepsState = { open: FormStep | null; done: ReadonlySet<FormStep> };

export const ADD_STEPS: StepsState = { open: "app", done: new Set() };
export const EDIT_STEPS: StepsState = { open: null, done: new Set(FORM_STEPS) };

export function stepStatus(state: StepsState, step: FormStep): StepStatus {
  if (state.open === step) {
    return "open";
  }
  return state.done.has(step) ? "done" : "todo";
}

export function completeStep(state: StepsState): StepsState {
  if (state.open === null) {
    return state;
  }
  const done = new Set(state.done).add(state.open);
  return { open: FORM_STEPS.find((step) => !done.has(step)) ?? null, done };
}

export function openStep(state: StepsState, step: FormStep): StepsState {
  const done = new Set(state.done);
  done.delete(step);
  return { open: step, done };
}

export function allStepsDone(state: StepsState): boolean {
  return FORM_STEPS.every((step) => state.done.has(step));
}
