"use client";
import { Button, EmptyHero, useStepWizard } from "@unkey/ui";
import {
  IconChevronLeftOutline18,
  IconCloudUploadOutline18,
  IconHardDriveOutline18,
  IconHeartPulseOutline18,
  IconLocation2Outline18,
  IconNodes2Outline18,
} from "nucleo-ui-outline-18";
import type { ReactNode } from "react";

type OnboardingStepHeaderProps = {
  title: ReactNode;
  subtitle?: ReactNode;
  showIconRow?: boolean;
  allowBack?: boolean;
};

export const OnboardingStepHeader = ({
  title,
  subtitle,
  showIconRow,
  allowBack,
}: OnboardingStepHeaderProps) => {
  const { back } = useStepWizard();

  return (
    <div className="flex flex-col items-center">
      {showIconRow && (
        <EmptyHero.Icons className="mb-0">
          <IconHardDriveOutline18 />
          <IconLocation2Outline18 />
          <IconCloudUploadOutline18 />
          <IconHeartPulseOutline18 />
          <IconNodes2Outline18 />
        </EmptyHero.Icons>
      )}
      {allowBack && (
        <Button
          variant="ghost"
          type="button"
          onClick={back}
          className="absolute top-3 left-3 z-50 flex items-center gap-1 hover:text-gray-11 group text-[13px] transition-colors text-gray-10"
        >
          <IconChevronLeftOutline18 className="! group-hover:text-gray-11" />
          Back
        </Button>
      )}
      <div className="flex flex-col items-center justify-center gap-2">
        <div className="font-semibold text-lg text-gray-12">{title}</div>
        {subtitle && <div className="text-[13px] text-gray-11 text-center">{subtitle}</div>}
      </div>
    </div>
  );
};
