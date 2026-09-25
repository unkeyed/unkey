"use client";
// biome-ignore lint/correctness/noUnusedImports: the package compiles JSX with the classic runtime ("jsx": "react"), so React must be in scope.
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
    <ShadcnDialogHeader className={cn("border-b bg-raised", className)}>
      <ShadcnDialogTitle className="px-6 py-4 text-gray-12 font-medium text-base flex flex-col">
        <span className="leading-8">{title}</span>
        {subTitle && <span className="text-gray-9 leading-5 text-sm font-normal">{subTitle}</span>}
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
        "bg-background flex flex-col gap-4 py-4 px-6 text-gray-11 overflow-y-auto scrollbar-hide grow",
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
    <ShadcnDialogFooter className={cn("p-6 border-t bg-raised text-gray-9", className)}>
      {children}
    </ShadcnDialogFooter>
  );
};
