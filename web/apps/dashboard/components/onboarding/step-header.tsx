"use client";
import {
  IconCloudUploadOutline18,
  IconHardDriveOutline18,
  IconHeartPulseOutline18,
  IconLocation2Outline18,
  IconNodes2Outline18,
} from "@unkey/icons";
import { IconFanRow } from "@unkey/ui";
import type { ReactNode } from "react";

type OnboardingStepHeaderProps = {
  title: ReactNode;
  subtitle?: ReactNode;
  showIconRow?: boolean;
};

export const OnboardingStepHeader = ({
  title,
  subtitle,
  showIconRow,
}: OnboardingStepHeaderProps) => {
  return (
    <div className="flex flex-col items-center">
      {showIconRow && (
        <IconFanRow className="mb-0">
          <IconHardDriveOutline18 />
          <IconLocation2Outline18 />
          <IconCloudUploadOutline18 />
          <IconHeartPulseOutline18 />
          <IconNodes2Outline18 />
        </IconFanRow>
      )}
      <div className="flex flex-col items-center justify-center gap-2">
        <div className="font-semibold text-lg text-gray-12">{title}</div>
        {subtitle && <div className="text-sm text-gray-11 text-center">{subtitle}</div>}
      </div>
    </div>
  );
};
