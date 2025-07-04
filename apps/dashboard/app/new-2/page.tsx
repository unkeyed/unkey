"use client";

import { ChevronLeft, StackPerspective2 } from "@unkey/icons";
import { Button, Separator } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

export default function OnboardingPage() {
  return (
    <div className="flex flex-col items-center justify-center pt-6">
      {/* Unkey Logo */}
      <div className="text-2xl font-medium text-gray-12 leading-7">Unkey</div>
      {/* Spacer */}
      <div className="mt-[72px]" />
      {/* Onboarding part. This will be a step wizard*/}
      <div className="flex flex-col">
        {/* Explanation part */}
        <div className="flex flex-col items-center">
          <div className="bg-grayA-3 rounded-full w-fit">
            <span className="px-3 text-xs leading-6 text-gray-12 font-medium">
              Step 1 of 3
            </span>
          </div>
          <div className="mt-5" />
          <div className="text-gray-12 font-semibold text-lg leading-8 ">
            Create company workspace
          </div>
          <div className="mt-2" />
          <div className="text-gray-9 font-normal text-[13px] leading-6 text-center w-3/4">
            Customize your workspace name, logo, and handle. This is how it’ll
            appear in your dashboard and URLs.
          </div>
        </div>
        <div className="mt-10" />
        {/* Form part */}
        <div className="border-gray-5 border rounded-2xl ">
          {/* Navigation part */}
          <div className="pl-2 pr-[14px] py-3 h-10 bg-gray-2 rounded-t-[15px] flex items-center">
            {/* Back button and current step name*/}
            <div className="flex items-center gap-3">
              <Button
                className="rounded-lg border-none bg-grayA-3 hover:bg-grayA-4 h-[22px]"
                variant="outline"
              >
                <div className="flex items-center gap-1">
                  <ChevronLeft
                    size="sm-regular"
                    className="text-gray-12 !w-3 !h-3 flex-shrink-0"
                  />
                  <span className="font-medium text-gray-12 text-xs">Back</span>
                </div>
              </Button>
              <div className="gap-[10px] items-center justify-center flex">
                <StackPerspective2 size="sm-regular" className="text-gray-11" />
                <span className="text-gray-12 font-medium text-xs">
                  Workspace
                </span>
              </div>
            </div>
            {/* Form validation progress circle*/}
            <div className="items-center flex gap-[10px] ml-auto">
              <div className="items-center flex gap-1.5 text-xs">
                <span className="font-medium text-gray-12">0</span>
                <span className="text-gray-10">of</span>
                <span className="font-medium text-gray-12">2</span>
                <span className="text-gray-10">required fields</span>
              </div>
              <div className="flex-shrink-0">
                <CircleProgress
                  value={2}
                  total={2}
                  size={12}
                  strokeWidth={1.5}
                />
              </div>
            </div>
          </div>
          <div className="border-t border-gray-5 p-10">
            <div>Arbitrary form section</div>

            <div className="mt-8" />
            <Separator className="my-2" />
            <div className="mb-8" />
            <Button size="xlg" className="w-full rounded-lg">
              Continue
            </Button>
            <div className="text-gray-9 leading-6 text-xs text-center mt-2">
              Next: you’ll create your first API key
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

type CircleProgressProps = {
  value: number;
  total: number;
  size?: number;
  strokeWidth?: number;
  className?: string;
};

const CircleProgress = ({
  value,
  total,
  size = 20,
  strokeWidth = 2,
  className = "",
}: CircleProgressProps) => {
  const isComplete = value >= total;

  if (isComplete) {
    return (
      <div className={cn("inline-flex items-center justify-center", className)}>
        <svg
          width={size}
          height={size}
          viewBox="0 0 18 18"
          className="text-success-9"
        >
          <g fill="currentColor" strokeLinecap="butt" strokeLinejoin="miter">
            <path
              d="M9 1.5a7.5 7.5 0 1 0 0 15 7.5 7.5 0 1 0 0-15z"
              fill="none"
              stroke="currentColor"
              strokeLinecap="square"
              strokeMiterlimit="10"
              strokeWidth={strokeWidth}
            />
            <path
              d="M5.25 9.75l2.25 2.25 5.25-6"
              fill="none"
              stroke="currentColor"
              strokeLinecap="square"
              strokeMiterlimit="10"
              strokeWidth={strokeWidth}
            />
          </g>
        </svg>
      </div>
    );
  }

  const radius = (size - strokeWidth) / 2;
  const circumference = radius * 2 * Math.PI;
  const progress = Math.min((value / total) * 100, 100);
  const strokeDasharray = circumference;
  const strokeDashoffset = circumference - (progress / 100) * circumference;

  return (
    <div className={cn("inline-flex items-center justify-center", className)}>
      <svg width={size} height={size} className="transform -rotate-90">
        {/* Background circle */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          className="text-gray-6"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          fill="none"
          opacity={0.3}
        />
        {/* Progress circle */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          className="text-success-9"
          strokeWidth={strokeWidth}
          stroke="currentColor"
          fill="none"
          strokeDasharray={strokeDasharray}
          strokeDashoffset={strokeDashoffset}
          strokeLinecap="round"
          style={{
            transition: "stroke-dashoffset 0.3s ease-in-out",
          }}
        />
      </svg>
    </div>
  );
};
