"use client";
import { ChevronLeft, CloudUp, Harddrive, HeartPulse, Location2, Nodes2 } from "@unkey/icons";
import { Button, useStepWizard } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";

type IconBoxProps = {
  children?: ReactNode;
  large?: boolean;
  className?: string;
};

const IconBox = ({ children, large, className }: IconBoxProps) => (
  <div
    className={cn(
      "shrink-0 flex items-center justify-center rounded-[10px] bg-transparent ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none",
      large ? "size-16" : "size-9",
      className,
    )}
  >
    {children}
  </div>
);

const iconItems: { icon: ReactNode; large?: boolean; opacity: string }[] = [
  { icon: null, opacity: "opacity-60" },
  { icon: <Harddrive className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <Location2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <CloudUp className="size-9" iconSize="md-thin" />, large: true, opacity: "opacity-90" },
  { icon: <HeartPulse className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <Nodes2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: null, opacity: "opacity-60" },
];

const IconRow = () => (
  <div
    className="p-2"
    style={{
      maskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
      WebkitMaskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
    }}
  >
    <div className="flex gap-6 items-center justify-center text-gray-12">
      {iconItems.map((item, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: its okay to use index
        <IconBox key={i} large={item.large} className={item.opacity}>
          {item.icon}
        </IconBox>
      ))}
    </div>
  </div>
);

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
      {showIconRow && <IconRow />}
      <div className="mb-5" />
      {allowBack && (
        <Button
          variant="ghost"
          type="button"
          onClick={back}
          className="absolute top-3 left-3 z-50 flex items-center gap-1 hover:text-gray-11 group text-[13px] transition-colors text-gray-10"
        >
          <ChevronLeft className="!size-3 group-hover:text-gray-11" iconSize="sm-regular" />
          Back
        </Button>
      )}
      <div className="flex flex-col items-center justify-center gap-2">
        <div className="font-semibold text-lg text-gray-12">{title}</div>
        {subtitle && <div className="text-[13px] text-gray-11 text-center">{subtitle}</div>}
      </div>
      <div className="mb-6" />
    </div>
  );
};
