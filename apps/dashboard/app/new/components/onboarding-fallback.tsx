"use client";
import { StackPerspective2 } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { OnboardingWizard } from "./onboarding-wizard";

export function OnboardingFallback() {
  return (
    <div className="h-screen flex flex-col items-center pt-6 overflow-hidden">
      {/* Unkey Logo */}
      <div className="text-2xl font-medium text-gray-12 leading-7">Unkey</div>
      {/* Spacer */}
      <div className="mt-[72px]" />
      {/* Static content while loading */}
      <div className="flex flex-col w-full max-w-sm sm:max-w-md lg:max-w-lg xl:max-w-xl">
        <div className="flex flex-col items-center h-[140px] justify-start">
          <div className="bg-grayA-3 rounded-full w-fit">
            <span className="px-3 text-xs leading-6 text-gray-12 font-medium tabular-nums">
              Step 1 of 3
            </span>
          </div>
          <div className="mt-5" />
          <div className="text-gray-12 font-semibold text-lg leading-8 text-center h-8 flex items-center">
            Create workspace
          </div>
          <div className="mt-2" />
          <div className="text-gray-9 font-normal text-[13px] leading-6 text-center px-4 h-[60px] flex items-start overflow-hidden">
            Set up your workspace to get started with Unkey
          </div>
        </div>
        <div className="mt-10" />
        <div className="flex-1 min-h-0">
          <OnboardingWizard
            steps={[
              {
                name: "Workspace",
                icon: <StackPerspective2 iconSize="sm-regular" className="text-gray-11" />,
                body: (
                  <form>
                    <div className="flex flex-col">
                      <div className="space-y-4 p-1">
                        <FormInput
                          value="Acme Corp"
                          placeholder="Enter workspace name"
                          label="Workspace name"
                          required
                          disabled
                        />
                      </div>
                    </div>
                  </form>
                ),
                kind: "required" as const,
                validFieldCount: 0,
                requiredFieldCount: 1,
                buttonText: "Continue",
                description: "Set up your workspace to get started",
                onStepNext: () => {},
                onStepBack: () => {},
              },
            ]}
            onComplete={() => {}}
            onStepChange={() => {}}
          />
        </div>
      </div>
    </div>
  );
}
