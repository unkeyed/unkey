"use client";
import { Refresh3, StackPerspective2, Trash } from "@unkey/icons";
import { useState } from "react";
import {
  type OnboardingStep,
  OnboardingWizard,
} from "./components/onboarding-wizard";
import { stepInfos } from "./constants";
import { Button, FormInput } from "@unkey/ui";

export default function OnboardingPage() {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);

  const steps: OnboardingStep[] = [
    {
      name: "Workspace",
      icon: <StackPerspective2 size="sm-regular" className="text-gray-11" />,
      body: (
        <div className="flex flex-col">
          <div className="flex flex-row py-1.5 gap-[18px]">
            <div className="bg-gradient-to-br from-info-5 to-blue-9 size-20 border rounded-2xl border-grayA-6" />
            <div className="flex flex-col gap-2">
              <div className="text-gray-11 text-[13px] leading-6">
                Company workspace logo
              </div>
              <div className="flex items-center gap-2">
                <Button variant="outline" className="w-fit">
                  <div className="gap-2 flex items-center text-[13px] leading-4 font-medium">
                    <Refresh3
                      className="text-gray-12 !w-3 !h-3 flex-shrink-0"
                      size="sm-regular"
                    />
                    Upload
                  </div>
                </Button>
                <Button variant="outline" className="w-fit">
                  <div className="gap-2 flex items-center text-[13px] leading-4 font-medium">
                    <Refresh3
                      className="text-gray-12 !w-3 !h-3 flex-shrink-0"
                      size="sm-regular"
                    />
                    Gradient
                  </div>
                </Button>
                <Trash size="md-regular" className="text-gray-8 ml-[10px]" />
              </div>
              <div className="text-gray-9 text-xs leading-6">
                .png, .jpg, or .svg up to 10MB, and 480×480px
              </div>
            </div>
          </div>
          <div className="space-y-4 pt-7">
            <FormInput
              placeholder="Enter workspace name"
              label="Workspace name"
              required
            />

            <FormInput
              placeholder="app.unkey.com/enter-a-handle"
              label="Workspace URL handleComplete"
              required
            />
          </div>
        </div>
      ),
      kind: "required" as const,
      filledInputCount: 0,
      totalInputCount: 2,
      onStepNext: (stepIndex) => {
        //TODO: Call validations
        console.log(`Leaving step ${stepIndex}`);
      },
      onStepBack: (stepIndex) => {
        //TODO: Call validations
        console.log(`Going back from step ${stepIndex}`);
      },
      description: "Next: you’ll create your first API key",
      buttonText: "Continue",
    },
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
    console.log("Onboarding completed!");
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
