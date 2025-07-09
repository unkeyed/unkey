"use client";
import { ConfirmPopover } from "@/components/confirmation-popover";
import { CircleInfo, Key2, StackPerspective2 } from "@unkey/icons";
import { Code, CopyButton, FormInput, VisibleButton } from "@unkey/ui";
import { Suspense, useRef, useState } from "react";
import { SecretKey } from "../(app)/apis/[apiId]/_components/create-key/components/secret-key";
import { type OnboardingStep, OnboardingWizard } from "./components/onboarding-wizard";
import { stepInfos } from "./constants";
import { useKeyCreationStep } from "./hooks/use-key-creation-step";
import { useWorkspaceStep } from "./hooks/use-workspace-step";

export default function OnboardingPage() {
  return (
    <Suspense fallback={<OnboardingFallback />}>
      <OnboardingContent />
    </Suspense>
  );
}

function OnboardingFallback() {
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
                icon: <StackPerspective2 size="sm-regular" className="text-gray-11" />,
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

function OnboardingContent() {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const workspaceStep = useWorkspaceStep();
  const keyCreationStep = useKeyCreationStep();

  const steps: OnboardingStep[] = [
    workspaceStep,
    keyCreationStep,
    {
      name: "API key",
      icon: <Key2 size="sm-regular" className="text-gray-11" />,
      body: (
        <OnboardingSuccessStep isConfirmOpen={isConfirmOpen} setIsConfirmOpen={setIsConfirmOpen} />
      ),
      kind: "non-required" as const,
      description: "You're all set! Your workspace and API key are ready",
      buttonText: "Continue to dashboard",
      onStepNext: () => {
        setIsConfirmOpen(true);
      },
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
    <div className="h-screen flex flex-col items-center pt-6 overflow-hidden">
      {/* Unkey Logo */}
      <div className="text-2xl font-medium text-gray-12 leading-7">Unkey</div>
      {/* Spacer */}
      <div className="mt-[72px]" />
      {/* Onboarding part. This will be a step wizard*/}
      <div className="flex flex-col w-full max-w-sm sm:max-w-md lg:max-w-lg xl:max-w-xl ">
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
        <div className="flex-1 min-h-0">
          <OnboardingWizard
            steps={steps}
            onComplete={handleComplete}
            onStepChange={handleStepChange}
          />
        </div>
      </div>
    </div>
  );
}

type OnboardingSuccessStepProps = {
  isConfirmOpen: boolean;
  setIsConfirmOpen: (open: boolean) => void;
};

const OnboardingSuccessStep = ({ isConfirmOpen, setIsConfirmOpen }: OnboardingSuccessStepProps) => {
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);
  const anchorRef = useRef<HTMLDivElement>(null);

  const keyData = { key: "key_data" };
  const apiId = "api_id";
  const split = keyData.key.split("_") ?? [];
  const maskedKey =
    split.length >= 2
      ? `${split.at(0)}_${"*".repeat(split.at(1)?.length ?? 0)}`
      : "*".repeat(split.at(0)?.length ?? 0);

  const snippet = `curl -XPOST '${
    process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"
  }/v1/keys.verifyKey' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "key": "${keyData.key}",
    "apiId": "${apiId}"
  }'`;

  return (
    <>
      <div>
        <span className="text-gray-11 text-[13px] leading-6" ref={anchorRef}>
          Run this command to verify your new API key against the API ID. This ensures your key is
          ready for authenticated requests.
        </span>
        <div className="flex flex-col gap-2 items-start w-full mt-6">
          <div className="text-gray-12 text-sm font-medium">Key Secret</div>
          <SecretKey value={keyData.key} title="API Key" className="bg-gray-2" />
          <div className="text-gray-9 text-[13px] flex items-center gap-1.5">
            <CircleInfo className="text-accent-9" size="sm-regular" />
            <span>
              Copy and save this key secret as it won't be shown again.{" "}
              <a
                href="https://www.unkey.com/docs/security/recovering-keys"
                target="_blank"
                rel="noopener noreferrer"
                className="text-info-11 hover:underline"
              >
                Learn more
              </a>
            </span>
          </div>
        </div>
        <div className="flex flex-col gap-2 items-start w-full mt-8">
          <div className="text-gray-12 text-sm font-medium">Try It Out</div>
          <Code
            className="bg-gray-2"
            visibleButton={
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
            }
            copyButton={<CopyButton value={snippet} />}
          >
            {showKeyInSnippet ? snippet : snippet.replace(keyData.key, maskedKey)}
          </Code>
        </div>
      </div>
      <ConfirmPopover
        isOpen={isConfirmOpen}
        onOpenChange={setIsConfirmOpen}
        onConfirm={() => setIsConfirmOpen(false)}
        triggerRef={anchorRef}
        title="You won't see this secret key again!"
        description="Make sure to copy your secret key before closing. It cannot be retrieved later."
        confirmButtonText="Close anyway"
        cancelButtonText="Dismiss"
        variant="warning"
        popoverProps={{
          side: "right",
          align: "end",
          sideOffset: 5,
          alignOffset: 30,
          onOpenAutoFocus: (e) => e.preventDefault(),
        }}
      />
    </>
  );
};
