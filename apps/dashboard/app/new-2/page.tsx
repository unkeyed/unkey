"use client";
import { StackPerspective2 } from "@unkey/icons";
import { useState } from "react";
import { type OnboardingStep, OnboardingWizard } from "./components/onboarding-wizard";
import { stepInfos } from "./constants";
import { useWorkspaceStep } from "./hooks/use-workspace-step";

export default function OnboardingPage() {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);

  const workspaceStep = useWorkspaceStep();
  const steps: OnboardingStep[] = [
    workspaceStep,
    {
      name: "API Key",
      icon: <StackPerspective2 size="sm-regular" className="text-gray-11" />,
      body: <div>API Key form content</div>,
      kind: "required" as const,
      filledInputCount: 0,
      totalInputCount: 1,
      description: "Next: you’ll create your first API key",
      buttonText: "Continue",
    },
    {
      name: "Dashboard",
      icon: <StackPerspective2 size="sm-regular" className="text-gray-11" />,
      body: <div>Dashboard setup content</div>,
      kind: "non-required" as const,
      description: "Next: you’ll create your first API key",
      buttonText: "Continue",
    },
  ];

  const handleComplete = () => {
    console.info("Onboarding completed!");
  };

  const handleStepChange = (newStepIndex: number) => {
    setCurrentStepIndex(newStepIndex);
  };

  const currentStepInfo = stepInfos[currentStepIndex];

  return (
    <div className="flex flex-col items-center justify-center pt-6">
      {/* Unkey Logo */}
      <div className="text-2xl font-medium text-gray-12 leading-7">Unkey</div>
      {/* Spacer */}
      <div className="mt-[72px]" />
      {/* Onboarding part. This will be a step wizard*/}
      <div className="flex flex-col w-full max-w-sm sm:max-w-md lg:max-w-lg xl:max-w-xl">
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
        <OnboardingWizard
          steps={steps}
          onComplete={handleComplete}
          onStepChange={handleStepChange}
        />
      </div>
    </div>
  );
}
