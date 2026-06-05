"use client";
import { ChevronLeft, CloudUp, Harddrive, HeartPulse, Location2, Nodes2 } from "@unkey/icons";
import { Button, useStepWizard } from "@unkey/ui";
import type { ReactNode } from "react";
import { ClassicIconRow } from "../_components/classic-icon-row";
import { ArtStyleSwitcher, useArtStyle } from "../_components/use-art-style";
import { IconHalftone } from "./icon-halftone";

// Flanked composition (small · large · small), every element a static halftone glyph.
// Side glyphs are smaller (fewer cols) and dimmer; the cloud is the focal point.
const rowItems: { icon: ReactNode; cols: number; className: string }[] = [
  { icon: <Harddrive iconSize="2xl-thin" />, cols: 18, className: "opacity-50" },
  { icon: <Location2 iconSize="2xl-thin" />, cols: 22, className: "opacity-75" },
  { icon: <CloudUp iconSize="2xl-thin" />, cols: 40, className: "" },
  { icon: <HeartPulse iconSize="2xl-thin" />, cols: 22, className: "opacity-75" },
  { icon: <Nodes2 iconSize="2xl-thin" />, cols: 18, className: "opacity-50" },
];

const IconRow = () => (
  <div
    className="p-2"
    style={{
      maskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
      WebkitMaskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
    }}
  >
    <div className="flex gap-5 items-center justify-center">
      {rowItems.map((item, i) => (
        <IconHalftone
          // biome-ignore lint/suspicious/noArrayIndexKey: static row, index is stable
          key={i}
          icon={item.icon}
          cols={item.cols}
          className={item.className}
        />
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
  const [style] = useArtStyle();

  return (
    <div className="flex flex-col items-center">
      <ArtStyleSwitcher />
      {showIconRow && (style === "ascii" ? <IconRow /> : <ClassicIconRow />)}
      {allowBack && (
        <Button
          variant="ghost"
          type="button"
          onClick={back}
          className="absolute top-3 left-3 z-50 flex items-center gap-1 hover:text-gray-11 group text-[13px] transition-colors text-gray-10"
        >
          <ChevronLeft className="size-3! group-hover:text-gray-11" iconSize="sm-regular" />
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
