"use client";
// biome-ignore lint/correctness/noUnusedImports: the package compiles JSX with the classic runtime ("jsx": "react"), so React must be in scope.
import * as React from "react";
import type { PropsWithChildren } from "react";
import { cn } from "../../../lib/utils";
import {
  DialogDescription as ShadcnDialogDescription,
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
    <ShadcnDialogHeader
      className={cn("space-y-0 border-b bg-raised px-4.5 pt-3.75 pb-4 pr-12", className)}
    >
      <ShadcnDialogTitle className="text-[1rem] leading-6">{title}</ShadcnDialogTitle>
      {subTitle && <ShadcnDialogDescription>{subTitle}</ShadcnDialogDescription>}
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
        "bg-raised flex flex-col gap-4 px-4.5 py-4 text-gray-11 overflow-y-auto scrollbar-hide grow",
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
      className={cn("px-4.5 py-4 border-t bg-background dark:bg-transparent", className)}
    >
      {children}
    </ShadcnDialogFooter>
  );
};
