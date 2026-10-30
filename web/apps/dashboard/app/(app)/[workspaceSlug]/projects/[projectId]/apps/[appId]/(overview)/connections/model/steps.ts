import type { ConnectionFormValues } from "./form";

export const FORM_STEPS = ["app", "config", "code"] as const;

export type FormStep = (typeof FORM_STEPS)[number];

export type StepStatus = "open" | "done" | "todo";

export type StepsState = { open: FormStep | null; done: ReadonlySet<FormStep> };

export const STEP_FIELDS: Record<FormStep, readonly (keyof ConnectionFormValues)[]> = {
  app: ["targetAppId"],
  config: ["name", "target"],
  code: [],
};

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

export function pickedApp(state: StepsState, appChanged: boolean): StepsState {
  const next = completeStep(state);
  return appChanged ? openStep(next, "config") : next;
}

export function firstStepWithError(
  errors: Partial<Record<keyof ConnectionFormValues, unknown>>,
): FormStep | null {
  return (
    FORM_STEPS.find((step) => STEP_FIELDS[step].some((field) => errors[field] !== undefined)) ??
    null
  );
}
