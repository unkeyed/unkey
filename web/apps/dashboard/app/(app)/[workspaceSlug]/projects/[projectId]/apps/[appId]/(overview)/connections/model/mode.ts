import type { StepsState } from "./steps";

export type FormMode = "add" | "edit";

export const FORM_MODES: Record<
  FormMode,
  {
    title: string;
    initialSteps: StepsState;
    appLocked: boolean;
    commitNeedsChanges: boolean;
    commitLabel: string;
    redeployNote: string | null;
  }
> = {
  add: {
    title: "Add connection",
    initialSteps: { open: "app", done: new Set() },
    appLocked: false,
    commitNeedsChanges: false,
    commitLabel: "Add connection",
    redeployNote: null,
  },
  edit: {
    title: "Edit connection",
    initialSteps: { open: "code", done: new Set(["app", "config"]) },
    appLocked: true,
    commitNeedsChanges: true,
    commitLabel: "Save changes",
    redeployNote: "Changes take effect after a redeploy.",
  },
};
