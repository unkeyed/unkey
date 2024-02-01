"use client";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import * as SliderPrimitive from "@radix-ui/react-slider";
import * as React from "React";
import { HelpCircle, KeySquare, ListChecks } from "lucide-react";
import { useState } from "react";

import {
  Asterisk,
  Bullet,
  Color,
  Cost,
  PricingCard,
  PricingCardContent,
  PricingCardFooter,
  PricingCardHeader,
  ProCardHighlight,
  Separator,
} from "./components";

const activeKeysSteps = [250, 1_000, 2_000, 5_000, 10_000, 50_000, 100_000, null];

const verificationsSteps = [
  150_000,
  250_000,
  500_000,
  1_000_000,
  10_000_000,
  100_000_000,
  1_000_000_000,
  null,
];

export const Discover: React.FC = () => {
  const [activeKeysIndex, setActiveKeysIndex] = useState(0);
  const activeKeys = activeKeysSteps.at(activeKeysIndex) ?? 0;
  const billableKeys = activeKeys - 250;
  const activeKeysCost = Math.max(0, billableKeys * 0.1);
  const [verificationsIndex, setVerificationsIndex] = useState(1_000_000);
  const verifications = verificationsSteps.at(verificationsIndex) ?? 0;
  const billableVerifications = verifications - 150_000;
  const verificationsCost = Math.max(0, (billableVerifications / 100_000) * 10);

  const totalCost = 25 + activeKeysCost + verificationsCost;

  return (
    <PricingCard color={Color.White} className="max-w-4xl mx-auto">
      <TooltipProvider delayDuration={10}>
        <PricingCardHeader
          title="Estimated cost calculator"
          description="Find out how much you will pay by using Unkey"
          withIcon={false}
          color={Color.Purple}
          className="bg-gradient-to-tr from-transparent to-[#ffffff]/10 "
        />
        <Separator />
        <PricingCardContent>
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <Cost dollar={fmtDollar(totalCost)} />
              <Tooltip>
                <TooltipTrigger asChild>
                  <HelpCircle className="w-4 h-4 text-white/40" style={{ strokeWidth: "1px" }} />
                </TooltipTrigger>
                <TooltipContent className="bg-black">
                  <p className="text-red-500">TODO</p>
                </TooltipContent>
              </Tooltip>
            </div>
            <p className="text-sm text-white/40">Resources are summed and billed monthly</p>
          </div>

          <div className="grid grid-cols-10 gap-4">
            <div className="cols-grid-1">
              <Bullet Icon={KeySquare} label="Active keys / month" color={Color.Purple} />
            </div>
            <div className="cols-grid-1">
              <Bullet
                Icon={ListChecks}
                label="Successful verifications / month"
                color={Color.Purple}
              />
            </div>
            <div className="flex flex-col items-center justify-between w-full h-full gap-8 border">
              <div className="flex items-center w-full h-10 border ">
                <Slider
                  min={0}
                  max={activeKeysSteps.length - 1}
                  value={[activeKeysIndex]}
                  className="w-full "
                  onValueChange={([v]) => setActiveKeysIndex(v)}
                />
              </div>
              <Slider
                min={0}
                max={verificationsSteps.length - 1}
                value={[verificationsIndex]}
                className="w-full"
                onValueChange={([v]) => setVerificationsIndex(v)}
              />
            </div>
            <div className="flex flex-col gap-8">
              <div className="flex items-center gap-2">
                <span className="text-white">{fmtNumber(activeKeys)}</span>
                <span className="text-white/40">Keys</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-white">{fmtNumber(verifications)}</span>
                <span className="text-white/40">Verifications</span>
              </div>
            </div>
            <div className="flex flex-col gap-8">
              <div className="flex items-center gap-2">
                <Asterisk tag={fmtDollar(activeKeysCost)} />
                <Tooltip>
                  <TooltipTrigger asChild>
                    <HelpCircle className="w-4 h-4 text-white/40" style={{ strokeWidth: "1px" }} />
                  </TooltipTrigger>
                  <TooltipContent className="bg-black">
                    <p className="text-red-500">TODO</p>
                  </TooltipContent>
                </Tooltip>
              </div>
              <div className="flex items-center gap-2">
                <Asterisk tag={fmtDollar(verificationsCost)} />
                <Tooltip>
                  <TooltipTrigger asChild>
                    <HelpCircle className="w-4 h-4 text-white/40" style={{ strokeWidth: "1px" }} />
                  </TooltipTrigger>
                  <TooltipContent className="bg-black">
                    <p className="text-red-500">TODO</p>
                  </TooltipContent>
                </Tooltip>
              </div>
            </div>
          </div>
        </PricingCardContent>
      </TooltipProvider>
    </PricingCard>
  );
};

function fmtNumber(n: number): string {
  return Intl.NumberFormat(undefined, { notation: "compact" }).format(n);
}

function fmtDollar(n: number): string {
  return Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(n);
}

const Slider = React.forwardRef<
  React.ElementRef<typeof SliderPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SliderPrimitive.Root>
>(({ className, ...props }, ref) => (
  <SliderPrimitive.Root
    ref={ref}
    className={cn("relative flex w-full touch-none select-none items-center", className)}
    {...props}
  >
    <SliderPrimitive.Track className="relative w-full h-px overflow-hidden rounded-full bg-gradient-to-r from-white/20 to-white/60 grow">
      <SliderPrimitive.Range className="absolute h-full bg-gradient-to-r from-[#02DEFC] via-[#0239FC] to-[#7002FC]" />
    </SliderPrimitive.Track>
    <SliderPrimitive.Thumb className="block w-4 h-4 transition-colors bg-white border-2 border-white rounded-full drop-shadow-[0_0_5px_rgba(255,255,255,1)]  focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50" />
  </SliderPrimitive.Root>
));
Slider.displayName = SliderPrimitive.Root.displayName;
