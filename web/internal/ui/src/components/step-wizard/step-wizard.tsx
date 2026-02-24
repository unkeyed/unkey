"use client";

import type { ReactNode } from "react";
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useReducer,
} from "react";
import { cn } from "../../lib/utils";
import type {
  StepKind,
  StepMeta,
  StepPosition,
  StepWizardContextValue,
  WizardAction,
  WizardState,
} from "./types";


function derivePosition(activeStepIndex: number, totalSteps: number): StepPosition {
  if (totalSteps === 0 || activeStepIndex < 0) return "empty";
  if (totalSteps === 1) return "only";
  if (activeStepIndex === 0) return "first";
  if (activeStepIndex === totalSteps - 1) return "last";
  return "middle";
}

function wizardReducer(state: WizardState, action: WizardAction): WizardState {
  switch (action.type) {
    case "REGISTER_STEP": {
      const { meta, defaultStepId } = action;

      const hasExistingOrder = state.insertionOrder.has(meta.id);
      const nextCounter = hasExistingOrder ? state.insertionCounter : state.insertionCounter + 1;
      const nextInsertionOrder = hasExistingOrder
        ? state.insertionOrder
        : new Map([...state.insertionOrder, [meta.id, state.insertionCounter]]);

      const exists = state.steps.some((s) => s.id === meta.id);
      const unsorted = exists
        ? state.steps.map((s) => (s.id === meta.id ? meta : s))
        : [...state.steps, meta];

      const sorted = [...unsorted].sort(
        (a, b) => (nextInsertionOrder.get(a.id) ?? 0) - (nextInsertionOrder.get(b.id) ?? 0),
      );

      const activeStepId =
        state.activeStepId === "" ? (defaultStepId ?? meta.id) : state.activeStepId;

      return {
        ...state,
        steps: sorted,
        activeStepId,
        insertionOrder: nextInsertionOrder,
        insertionCounter: nextCounter,
      };
    }

    case "UNREGISTER_STEP": {
      const nextSteps = state.steps.filter((s) => s.id !== action.id);
      const nextInsertionOrder = new Map(state.insertionOrder);
      nextInsertionOrder.delete(action.id);
      return {
        ...state,
        steps: nextSteps,
        insertionOrder: nextInsertionOrder,
      };
    }

    case "GO_NEXT": {
      const idx = state.steps.findIndex((s) => s.id === state.activeStepId);
      if (idx < 0 || idx >= state.steps.length - 1) return state;
      return { ...state, activeStepId: state.steps[idx + 1].id };
    }

    case "GO_BACK": {
      const idx = state.steps.findIndex((s) => s.id === state.activeStepId);
      if (idx <= 0) return state;
      const currentStep = state.steps[idx];
      if (currentStep.preventBack) return state;
      return { ...state, activeStepId: state.steps[idx - 1].id };
    }

    case "GO_TO": {
      if (!state.steps.some((s) => s.id === action.id)) return state;
      return { ...state, activeStepId: action.id };
    }
  }
}


const StepWizardContext = createContext<StepWizardContextValue | undefined>(undefined);

export const useStepWizard = (): StepWizardContextValue => {
  const context = useContext(StepWizardContext);
  if (context === undefined) {
    throw new Error("useStepWizard must be used within a StepWizard.Root");
  }
  return context;
};


type StepWizardRootProps = {
  onComplete?: () => void;
  defaultStepId?: string;
  className?: string;
  children: ReactNode;
};

const StepWizardRoot = ({
  onComplete,
  defaultStepId,
  className,
  children,
}: StepWizardRootProps) => {
  const [state, dispatch] = useReducer(wizardReducer, {
    steps: [],
    activeStepId: defaultStepId ?? "",
    insertionOrder: new Map(),
    insertionCounter: 0,
  });

  const activeStepIndex = state.steps.findIndex((s) => s.id === state.activeStepId);
  const totalSteps = state.steps.length;
  const position = derivePosition(activeStepIndex, totalSteps);
  const currentStep = activeStepIndex >= 0 ? state.steps[activeStepIndex] : undefined;

  const isFirstStep = position === "first" || position === "only";
  const isLastStep = position === "last" || position === "only";
  const canGoBack = !isFirstStep && !(currentStep?.preventBack ?? false);
  const canGoForward = position === "first" || position === "middle";

  const registerStep = useCallback(
    (meta: StepMeta) => dispatch({ type: "REGISTER_STEP", meta, defaultStepId }),
    [defaultStepId],
  );
  const unregisterStep = useCallback(
    (id: string) => dispatch({ type: "UNREGISTER_STEP", id }),
    [],
  );
  const back = useCallback(() => dispatch({ type: "GO_BACK" }), []);
  const goTo = useCallback((id: string) => dispatch({ type: "GO_TO", id }), []);

  const next = useCallback(() => {
    if (isLastStep) {
      onComplete?.();
      return;
    }
    dispatch({ type: "GO_NEXT" });
  }, [isLastStep, onComplete]);

  const skip = useCallback(() => {
    if (currentStep?.kind !== "optional") return;
    next();
  }, [currentStep, next]);

  const contextValue: StepWizardContextValue = {
    steps: state.steps,
    activeStepId: state.activeStepId,
    activeStepIndex,
    totalSteps,
    position,
    next,
    back,
    skip,
    goTo,
    registerStep,
    unregisterStep,
    canGoBack,
    canGoForward,
    isLastStep,
    isFirstStep,
  };

  return (
    <StepWizardContext.Provider value={contextValue}>
      <div className={cn("flex flex-col", className)}>{children}</div>
    </StepWizardContext.Provider>
  );
};


type StepWizardStepProps = {
  id: string;
  label: string;
  kind?: StepKind;
  preventBack?: boolean;
  children: ReactNode;
};

const StepWizardStep = ({
  id,
  label,
  kind = "required",
  preventBack,
  children,
}: StepWizardStepProps) => {
  const { registerStep, unregisterStep, activeStepId } = useStepWizard();

  useEffect(() => {
    registerStep({ id, label, kind, preventBack });
    return () => unregisterStep(id);
    // Only re-register when props that affect metadata change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, label, kind, preventBack]);

  const isActive = id === activeStepId;

  return (
    <div
      className={cn(
        "w-full absolute inset-0 overflow-y-auto scrollbar-hide",
        "transition-all duration-300 ease-out",
        isActive
          ? "opacity-100 translate-x-0 z-10"
          : "opacity-0 translate-x-5 z-0 pointer-events-none",
      )}
      aria-hidden={!isActive}
    >
      {children}
    </div>
  );
};

StepWizardRoot.displayName = "StepWizardRoot";
StepWizardStep.displayName = "StepWizardStep";

export const StepWizard = {
  Root: StepWizardRoot,
  Step: StepWizardStep,
};
