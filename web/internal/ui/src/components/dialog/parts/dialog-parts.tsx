"use client";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import type { PropsWithChildren } from "react";
import { cn } from "../../../lib/utils";
import {
  DialogFooter as ShadcnDialogFooter,
  DialogHeader as ShadcnDialogHeader,
  DialogTitle as ShadcnDialogTitle,
} from "../dialog";

type DefaultDialogHeaderProps = {
  title: string;
  subTitle?: string;
  className?: string;
};

export const DefaultDialogHeader = ({ title, subTitle, className }: DefaultDialogHeaderProps) => {
  return (
    <ShadcnDialogHeader className={cn("border-b border-gray-4 bg-white dark:bg-black", className)}>
      <ShadcnDialogTitle className="px-6 py-4 text-gray-12 font-medium text-base flex flex-col">
        <span className="leading-[32px]">{title}</span>
        {subTitle && ( // Conditionally render subtitle span only if it exists
          <span className="text-gray-9 leading-[20px] text-[13px] font-normal">{subTitle}</span>
        )}
      </ShadcnDialogTitle>
    </ShadcnDialogHeader>
  );
};

type DefaultDialogContentAreaProps = PropsWithChildren<{
  className?: string;
}>;

export const DefaultDialogContentArea = ({
  children,
  className,
}: DefaultDialogContentAreaProps) => {
  return (
    <div
      className={cn(
        "bg-grayA-2 flex flex-col gap-4 py-4 px-6 text-gray-11 overflow-y-auto scrollbar-hide flex-grow",
        className,
      )}
    >
      {children}
    </div>
  );
};

type DefaultDialogFooterProps = PropsWithChildren<{
  className?: string;
}>;

export const DefaultDialogFooter = ({ children, className }: DefaultDialogFooterProps) => {
  return (
    <ShadcnDialogFooter
      className={cn("p-6 border-t border-grayA-4 bg-white dark:bg-black text-gray-9", className)}
    >
      {children}
    </ShadcnDialogFooter>
  );
};
