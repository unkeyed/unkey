"use client";
import { HelpButton } from "@/components/navigation/sidebar/help-button";
import { UserButton } from "@/components/navigation/sidebar/user-button";
import { Key2 } from "@unkey/icons";
import { useState } from "react";
import { stepInfos } from "../constants";
import { useKeyCreationStep } from "../hooks/use-key-creation-step";
import { useWorkspaceStep } from "../hooks/use-workspace-step";
import { OnboardingSuccessStep } from "./onboarding-success-step";
import { type OnboardingStep, OnboardingWizard } from "./onboarding-wizard";

export function OnboardingContent() {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const workspaceStep = useWorkspaceStep();
  const keyCreationStep = useKeyCreationStep();

  const steps: OnboardingStep[] = [
    workspaceStep,
    keyCreationStep,
    {
      name: "API key",
      icon: <Key2 iconSize="sm-regular" className="text-gray-11" />,
      body: (
        <OnboardingSuccessStep isConfirmOpen={isConfirmOpen} setIsConfirmOpen={setIsConfirmOpen} />
      ),
      kind: "non-required" as const,
      description: "You're all set! Your workspace and API key are ready",
      buttonText: "Continue to dashboard",
      onStepNext: () => {
        setIsConfirmOpen(true);
      },
      onStepSkip: () => {
        setIsConfirmOpen(true);
      },
    },
  ];

  const handleStepChange = (newStepIndex: number) => {
    setCurrentStepIndex(newStepIndex);
  };

  const currentStepInfo = stepInfos[currentStepIndex];

  return (
    <div className="h-screen flex flex-col items-center pt-6 overflow-hidden relative">
      {/* Unkey Logo */}
      <div className="text-2xl font-medium text-gray-12 leading-7">Unkey</div>
      {/* Spacer */}
      <div className="mt-[72px]" />
      {/* Onboarding part. This will be a step wizard*/}
      <div className="flex flex-col w-full max-w-sm sm:max-w-md lg:max-w-lg xl:max-w-xl flex-1 min-h-0 ">
        {/* Explanation part - Fixed height to prevent layout shifts */}
        <div className="flex flex-col items-center h-[140px] justify-start">
          <div className="bg-grayA-3 rounded-full w-fit">
            <span className="px-3 text-xs leading-6 text-gray-12 font-medium tabular-nums">
              Step {currentStepIndex + 1} of {steps.length}
            </span>
          </div>
          <div className="mt-5" />
          <div className="text-gray-12 font-semibold text-lg leading-8 text-center h-8 flex items-center">
            {currentStepInfo.title}
          </div>
          <div className="mt-2" />
          <div className="text-gray-9 font-normal text-[13px] leading-6 text-center px-4 h-[60px] flex items-start overflow-hidden">
            {currentStepInfo.description}
          </div>
        </div>
        <div className="mt-10" />
        {/* Form part */}
        <div className="flex-1 min-h-0 overflow-y-auto overscroll-y-contain pb-[calc(6rem+env(safe-area-inset-bottom))]">
          <OnboardingWizard steps={steps} onStepChange={handleStepChange} />
        </div>
      </div>
      <div className="absolute bottom-4 left-4">
        <UserButton />
      </div>
      <div className="absolute bottom-4 right-4">
        <HelpButton />
      </div>
    </div>
  );
}
