import React from "react";

import { cn } from "../lib/utils";

interface EmptyRootProps extends React.HTMLAttributes<HTMLDivElement> {
  fill?: boolean;
}
export function Empty({ className, children, ...props }: EmptyRootProps) {
  return (
    <div
      className={cn("animate-in fade-in-50 p-8 text-center h-full w-full", className)}
      {...props}
    >
      <div className="flex flex-col items-center justify-center h-full w-full ">{children}</div>
    </div>
  );
}

Empty.Icon = function EmptyIcon({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-28 w-28 justify-center items-center">
      <div className="relative z-10 ">
        <div className="absolute top-0 left-1/2 -translate-x-1/2 bg-gradient-to-r from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-32 h-[1px] " />
        <div className="absolute bottom-0 left-1/2 -translate-x-1/2 bg-gradient-to-r from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-32 h-[1px]  " />
        <div className="absolute left-0 top-1/2 -translate-y-1/2 h-32 bg-gradient-to-t from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-[1px]" />
        <div className="absolute right-0 top-1/2 -translate-y-1/2 h-32 bg-gradient-to-t from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-[1px] " />
        <div className="flex w-16 h-16 items-center justify-center rounded-2xl bg-gray-2 border border-[1px] border-[hsla(240,100%,10%,0.06)] dark:border-[hsla(211,66%,92%,0.2)] text-accent-12 z-50">
          {children}
        </div>
      </div>
    </div>
  ); 
};

type EmptyTitleProps = React.HTMLAttributes<HTMLHeadingElement>;

Empty.Title = function EmptyTitle({ className, ...props }: EmptyTitleProps) {
  return (
    <h2
      className={cn("text-accent-12 mt-3 font-semibold text-[15px] leading-6", className)}
      {...props}
    />
  );
};

type EmptyDescriptionProps = React.HTMLAttributes<HTMLParagraphElement>;

Empty.Description = function EmptyDescription({ className, ...props }: EmptyDescriptionProps) {
  return (
    <p
      className={cn("text-accent-11 text-center text-xs font-normal leading-6 mt-1", className)}
      {...props}
    />
  );
};

type EmptyActionProps = React.HTMLAttributes<HTMLDivElement>;

Empty.Actions = function EmptyAction({ className, children, ...props }: EmptyActionProps) {
  return (
    <div
      className={cn("flex gap-4 mt-2 px-3 items-center justify-center leading-6", className)}
      {...props}
    >
      {children}
    </div>
  );
};
