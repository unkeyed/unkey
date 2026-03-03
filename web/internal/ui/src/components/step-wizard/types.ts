export type StepKind = "required" | "optional";

export type StepMeta = {
  id: string;
  label: string;
  kind: StepKind;
  preventBack?: boolean;
};

export type StepPosition = "empty" | "only" | "first" | "middle" | "last";

export type WizardState = {
  readonly steps: readonly StepMeta[];
  readonly activeStepId: string;
  readonly insertionOrder: ReadonlyMap<string, number>;
  readonly insertionCounter: number;
};

export type WizardAction =
  | { type: "REGISTER_STEP"; meta: StepMeta; defaultStepId: string | undefined }
  | { type: "UNREGISTER_STEP"; id: string }
  | { type: "GO_NEXT" }
  | { type: "GO_BACK" }
  | { type: "GO_TO"; id: string };

export type StepWizardContextValue = {
  /** Ordered step metadata */
  steps: readonly StepMeta[];
  /** Current active step ID */
  activeStepId: string;
  /** Current step index (derived) */
  activeStepIndex: number;
  /** Total step count */
  totalSteps: number;
  /** Semantic position of the active step */
  position: StepPosition;
  /** Navigate to next step */
  next: () => void;
  /** Navigate to previous step */
  back: () => void;
  /** Skip current step (only for optional steps) */
  skip: () => void;
  /** Navigate to a specific step by ID */
  goTo: (id: string) => void;
  /** Register a step (called by StepWizard.Step on mount) */
  registerStep: (meta: StepMeta) => void;
  /** Unregister a step (called on unmount) */
  unregisterStep: (id: string) => void;
  /** Whether the current step allows going back */
  canGoBack: boolean;
  /** Whether there is a next step */
  canGoForward: boolean;
  /** Whether the current step is the last */
  isLastStep: boolean;
  /** Whether the current step is the first */
  isFirstStep: boolean;
};
