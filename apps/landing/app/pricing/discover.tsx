"use client";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import * as SliderPrimitive from "@radix-ui/react-slider";
import { HelpCircle, KeySquare, ListChecks } from "lucide-react";
import React, { useState } from "react";

import { SectionTitle } from "../section-title";
import {
  Bullet,
  Color,
  Cost,
  FreeCardHighlight,
  PricingCard,
  PricingCardContent,
  PricingCardFooter,
  PricingCardHeader,
  Separator,
} from "./components";
import { SubDiscoverySvg } from "./svgs";

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
  const activeKeys = activeKeysSteps[activeKeysIndex];
  const billableKeys = Math.max(0, (activeKeys ?? 0) - 250);
  const activeKeysCost = billableKeys * 0.1;
  const activeKeysQuantityDisplay = fmtNumber(activeKeys ?? Number.POSITIVE_INFINITY);
  const activeKeysCostDisplay = activeKeys === null ? "Custom" : fmtDollar(activeKeysCost);

  const [verificationsIndex, setVerificationsIndex] = useState(0);
  const verifications = verificationsSteps[verificationsIndex];
  const billableVerifications = Math.max(0, (verifications ?? 0) - 150_000);
  const verificationsCost = (billableVerifications * 10) / 100_000;
  const verificationsQuantityDisplay = fmtNumber(verifications ?? Number.POSITIVE_INFINITY);
  const verificationsCostDisplay = verifications === null ? "Custom" : fmtDollar(verificationsCost);

  const totalCostDisplay =
    verifications === null || activeKeys === null
      ? "Custom"
      : fmtDollar(25 + activeKeysCost + verificationsCost);

  return (
    <div>
      <SectionTitle
        align="center"
        title={
          <>
            Discover your pricing. <br /> Pay only what matters to you
          </>
        }
        text={
          <>
            Find out exactly hwat your investment will be on Unkey, with our estimated cost
            calculator.
            <br />
            Explore the cost per active key and key verifications.
          </>
        }
      />

      <PricingCard color={Color.White} className="relative max-w-4xl mx-auto mt-20">
        <FreeCardHighlight className="absolute top-0 right-0" />
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
                <Cost dollar={totalCostDisplay} />
                <Tooltip>
                  <TooltipTrigger asChild>
                    <HelpCircle className="w-4 h-4 text-white/40" style={{ strokeWidth: "1px" }} />
                  </TooltipTrigger>
                  <TooltipContent className="bg-black">
                    <p className="text-sm text-white/40">Cost break down:</p>

                    <div className="grid grid-cols-2 mt-4 gap-x-4 gap-y-2">
                      <span className="text-white">{fmtDollar(25)}</span>
                      <span className="text-sm text-white/40">Base Plan</span>
                      <span className="text-white">{activeKeysCostDisplay}</span>
                      <span className="text-sm text-white/40">Active Keys</span>
                      <span className="text-white">{verificationsCostDisplay}</span>
                      <span className="text-sm text-white/40">Verifications</span>
                    </div>
                  </TooltipContent>
                </Tooltip>
              </div>
              <p className="text-sm text-white/40">Resources are summed and billed monthly</p>
            </div>

            <div className="flex flex-col gap-8">
              <Row
                label={<Bullet Icon={KeySquare} label="Active keys" color={Color.Purple} />}
                slider={
                  <Slider
                    min={0}
                    max={activeKeysSteps.length - 1}
                    value={[activeKeysIndex]}
                    className=""
                    onValueChange={([v]) => setActiveKeysIndex(v)}
                  />
                }
                quantity={
                  <div className="flex items-center gap-2">
                    <span className="text-white">{activeKeysQuantityDisplay}</span>
                    <span className="text-sm text-white/40">Keys</span>
                  </div>
                }
                cost={<PriceTag dollar={activeKeysCostDisplay} />}
              />
              <Row
                label={<Bullet Icon={ListChecks} label="Verifications" color={Color.Purple} />}
                slider={
                  <Slider
                    min={0}
                    max={verificationsSteps.length - 1}
                    value={[verificationsIndex]}
                    className=""
                    onValueChange={([v]) => setVerificationsIndex(v)}
                  />
                }
                quantity={
                  <div className="flex items-center gap-2">
                    <span className="text-white">{verificationsQuantityDisplay}</span>
                    <span className="text-sm text-white/40">Verifications</span>
                  </div>
                }
                cost={<PriceTag dollar={verificationsCostDisplay} />}
              />
            </div>
          </PricingCardContent>
          <PricingCardFooter />
        </TooltipProvider>
      </PricingCard>{" "}
      <SubDiscoverySvg className="container max-w-4xl mx-auto" />
    </div>
  );
};

const Row: React.FC<{
  label: React.ReactNode;
  slider: React.ReactNode;
  quantity: React.ReactNode;
  cost: React.ReactNode;
}> = ({ label, slider, quantity, cost }) => {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="w-2/12">{label}</div>
      <div className="w-6/12">{slider}</div>

      <div className="w-2/12">{quantity}</div>

      <div className="w-2/12">{cost}</div>
    </div>
  );
};

function fmtNumber(n: number): string {
  return Intl.NumberFormat(undefined, { notation: "compact" }).format(n);
}

function fmtDollar(n: number): string {
  return Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(n);
}

const PriceTag: React.FC<{ dollar: string }> = ({ dollar }) => {
  return (
    <div className="flex justify-end w-full">
      <span className="h-6 px-2 text-sm font-semibold text-white rounded bg-white/10">
        {dollar}
      </span>
    </div>
  );
};

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
