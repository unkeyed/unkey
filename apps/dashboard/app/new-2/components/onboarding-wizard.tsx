import { ChevronLeft, ChevronRight } from "@unkey/icons";
import { Button, Separator } from "@unkey/ui";
import { useEffect, useState } from "react";
import { CircleProgress } from "./circle-progress";

export type OnboardingStep = {
  /** Display name of the step shown in the navigation */
  name: string;
  /** Icon component displayed next to the step name */
  icon: JSX.Element;
  /** Main content/form rendered for this step */
  body: React.ReactNode;
  /** Callback fired when user clicks next/continue button */
  onStepNext?: (currentStep: number) => void;
  /** Callback fired when user clicks back button */
  onStepBack?: (currentStep: number) => void;
  /** Description text shown below the main button */
  description: string;
  /** Text displayed on the primary action button */
  buttonText: string;
} & (
  | {
      /** Step type - no validation required, can be skipped */
      kind: "non-required";
    }
  | {
      /** Step type - has form validation, must be completed */
      kind: "required";
      /** Number of fields that are valid (no errors and have values) */
      validFieldCount: number;
      /** Total number of required fields that must be completed */
      requiredFieldCount: number;
    }
);

export type OnboardingWizardProps = {
  /** Array of steps to display in the wizard. Must contain at least one step. */
  steps: OnboardingStep[];
  /** Callback fired when the wizard is completed (user clicks continue on last step) */
  onComplete?: () => void;
  /** Callback fired whenever the current step changes */
  onStepChange?: (stepIndex: number) => void;
};

const CIRCLE_PROGRESS_SIZE = 12;
const CIRCLE_PROGRESS_STROKE_WIDTH = 1.5;

export const OnboardingWizard = ({ steps, onComplete, onStepChange }: OnboardingWizardProps) => {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);

  if (steps.length === 0) {
    throw new Error("OnboardingWizard requires at least one step");
  }

  useEffect(() => {
    onStepChange?.(currentStepIndex);
  }, [currentStepIndex, onStepChange]);

  const currentStep = steps[currentStepIndex];
  const isFirstStep = currentStepIndex === 0;
  const isLastStep = currentStepIndex === steps.length - 1;

  const handleBack = () => {
    if (!isFirstStep) {
      currentStep.onStepBack?.(currentStepIndex);
      setCurrentStepIndex(currentStepIndex - 1);
    }
  };

  const handleNext = () => {
    if (isLastStep) {
      currentStep.onStepNext?.(currentStepIndex);
      onComplete?.();
    } else {
      currentStep.onStepNext?.(currentStepIndex);
      setCurrentStepIndex(currentStepIndex + 1);
    }
  };

  const handleSkip = () => {
    // For non-required steps, skip to next step
    if (isLastStep) {
      onComplete?.();
    } else {
      setCurrentStepIndex(currentStepIndex + 1);
    }
  };

  return (
    <div className="border-gray-5 border rounded-2xl flex flex-col h-auto max-h-[400px] sm:max-h-[500px] md:max-h-[600px] lg:max-h-[700px] xl:max-h-[800px]">
      {/* Navigation part */}
      <div className="pl-2 pr-[14px] py-3 h-10 bg-gray-2 rounded-t-[15px] flex items-center">
        {/* Back button and current step name*/}
        <div className="flex items-center gap-3">
          <Button
            className="rounded-lg bg-grayA-3 hover:bg-grayA-4 h-[22px]"
            variant="outline"
            onClick={handleBack}
            disabled={isFirstStep}
          >
            <div className="flex items-center gap-1">
              <ChevronLeft size="sm-regular" className="text-gray-12 !w-3 !h-3 flex-shrink-0" />
              <span className="font-medium text-gray-12 text-xs">Back</span>
            </div>
          </Button>
          <div className="gap-[10px] items-center justify-center flex">
            {currentStep.icon}
            <span className="text-gray-12 font-medium text-xs">{currentStep.name}</span>
          </div>
        </div>
        {/* Form validation progress circle*/}
        <div className="ml-auto">
          {currentStep.kind === "required" ? (
            <div className="items-center flex gap-[10px]">
              <div className="items-center flex gap-1.5 text-xs">
                <span className="font-medium text-gray-12">{currentStep.validFieldCount}</span>
                <span className="text-gray-10">of</span>
                <span className="font-medium text-gray-12">{currentStep.requiredFieldCount}</span>
                <span className="text-gray-10">required fields</span>
              </div>
              <div className="flex-shrink-0">
                <CircleProgress
                  value={currentStep.validFieldCount}
                  total={currentStep.requiredFieldCount}
                  size={CIRCLE_PROGRESS_SIZE}
                  strokeWidth={CIRCLE_PROGRESS_STROKE_WIDTH}
                />
              </div>
            </div>
          ) : (
            <div className="flex items-center gap-3">
              <span className="text-gray-10 text-xs">No required fields</span>
              <Button
                className="rounded-lg bg-grayA-3 hover:bg-grayA-4 h-[22px]"
                variant="outline"
                onClick={handleSkip}
              >
                <div className="flex items-center gap-1">
                  <span className="font-medium text-gray-12 text-xs">Skip step</span>
                  <ChevronRight
                    size="sm-regular"
                    className="text-gray-12 !w-3 !h-3 flex-shrink-0"
                  />
                </div>
              </Button>
            </div>
          )}
        </div>
      </div>
      <div className="border-t border-gray-5 p-10 flex flex-col flex-1 min-h-0">
        {/* Scrollable step content */}
        <div className="flex-1 overflow-y-auto min-h-0 scrollbar-hide">{currentStep.body}</div>

        <div className="mt-8" />
        <Separator className="my-2" />
        <div className="mb-8" />

        {/* Fixed footer */}
        <div className="flex-shrink-0">
          <Button
            size="xlg"
            className="w-full rounded-lg"
            onClick={handleNext}
            disabled={
              currentStep.kind === "required"
                ? currentStep.validFieldCount !== currentStep.requiredFieldCount
                : false
            }
          >
            {currentStep.buttonText}
          </Button>
          <div className="text-gray-9 leading-6 text-xs text-center mt-2">
            {currentStep.description}
          </div>
        </div>
      </div>
    </div>
  );
};
