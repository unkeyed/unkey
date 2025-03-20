import { ProgressCircle } from "@/app/(app)/settings/billing/components/usage";
import { createContext } from "@/lib/create-context";
import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import type { Workspace } from "@unkey/db";
import { Button } from "@unkey/ui";
import { motion } from "framer-motion";
import Link from "next/link";
import React from "react";

/* ----------------------------------------------------------------------------
 * UsageInsight - Root
 * --------------------------------------------------------------------------*/

type DivElement = React.ElementRef<typeof motion.div>;
type DivProps = React.ComponentPropsWithoutRef<typeof motion.div>;
type UsageContextValue = {
  tier: Workspace["tier"];
  current: number;
  max: number;
};
export interface UsageRootProps extends DivProps {
  tier: Workspace["tier"];
  current: number;
  max: number;
  isLoading?: boolean;
  children: React.ReactNode | React.ReactNode[];
}

const ROOT_NAME = "UsageRoot";

const [UsageProvider, useUsageContext] = createContext<UsageContextValue>(ROOT_NAME);

export const Root = React.forwardRef<DivElement, UsageRootProps>((props, ref) => {
  const { tier, current, max, className, children, isLoading = false, ...rootProps } = props;

  return (
    <UsageProvider tier={tier} current={current} max={max}>
      <motion.div
        {...rootProps}
        layout
        ref={ref}
        className={cn(
          "relative flex flex-col bg-background border border-border rounded-xl p-4 group overflow-hidden w-full",
          { "max-h-[102px]": isLoading, className },
        )}
      >
        {isLoading ? (
          <div className="z-10 flex flex-col w-full gap-3">
            <div className="h-5 w-20 bg-secondary/75 rounded-md animate-pulse" />
            <div className="flex gap-2">
              <div className="h-7 w-7 bg-secondary/75 rounded-md animate-pulse delay-100" />
              <div className="flex flex-col items-start justify-start gap-1.5">
                <div className="h-4 w-20 bg-secondary/75 rounded-md animate-pulse delay-150" />
                <div className="h-3 w-32 bg-secondary/50 rounded-md animate-pulse delay-200" />
              </div>
            </div>
          </div>
        ) : (
          <div className="z-10 flex flex-col w-full gap-3">
            <h2 className="text-gray-12 text-lg capitalize leading-5">{tier}</h2>

            {children}

            {current / max >= 0.95 && (
              <Button variant="primary" size="sm" className="w-full">
                <Link href="/settings/billing">Upgrade</Link>
              </Button>
            )}
          </div>
        )}
      </motion.div>
    </UsageProvider>
  );
});

Root.displayName = ROOT_NAME;

/* ----------------------------------------------------------------------------
 * UsageInsight - Details
 * --------------------------------------------------------------------------*/

const DETAILS_NAME = "UsageDetails";

export const Details = React.forwardRef<DivElement, DivProps>((props, ref) => {
  const { className, children, ...detailsProps } = props;

  return (
    <motion.div
      initial={{ opacity: 0, x: -16 }}
      animate={{ opacity: 1, x: -0 }}
      transition={{
        ease: "easeInOut",
        duration: 0.4,
        opacity: { duration: 0.8, delay: 0.1 },
      }}
      {...detailsProps}
      ref={ref}
      className={cn("duration-500 ease-out w-full flex flex-col items-start justify-start", {
        className,
      })}
    >
      {children}
    </motion.div>
  );
});

Details.displayName = DETAILS_NAME;

/* ----------------------------------------------------------------------------
 * UsageInsight - Item
 * --------------------------------------------------------------------------*/
type PrimitiveDivElement = React.ElementRef<"div">;
type PrimitiveDivProps = React.ComponentPropsWithoutRef<"div">;

export interface UsageItemProps extends PrimitiveDivProps {
  item?: {
    current: number;
    max: number;
  };
  title: string;
  description?: string;
  color?: string;
}

const ITEM_NAME = "UsageItem";

export const Item = React.forwardRef<PrimitiveDivElement, UsageItemProps>((props, ref) => {
  const { item, title, description, color, className, ...itemProps } = props;
  const { current, max } = useUsageContext("UsageItem");

  return (
    <div
      {...itemProps}
      ref={ref}
      className={cn("flex items-start justify-start gap-3 w-full", {
        className,
      })}
    >
      <ProgressCircle
        value={item?.current ?? current}
        max={item?.max ?? max}
        color={color ?? "orange"}
      />
      <div className="flex flex-col gap-1.5 justify-start items-start select-none">
        <h6 className="text-gray-12 text-sm leading-none">{title}</h6>
        <p className="text-xs font-normal line-clamp-1">
          {formatNumber(item?.current ?? current)} of {formatNumber(item?.max ?? max)} {description}
        </p>
      </div>
    </div>
  );
});

Item.displayName = ITEM_NAME;

/* -------------------------------------------------------------------------------------------------
 * Exports
 * -----------------------------------------------------------------------------------------------*/

export const UsageInsight = Object.assign(
  {},
  {
    Root,
    Details,
    Item,
  },
);
