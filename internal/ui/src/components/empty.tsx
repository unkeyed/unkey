// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import { cn } from "../lib/utils";
import { Ufo } from "@unkey/icons";

interface EmptyRootProps extends React.HTMLAttributes<HTMLDivElement> {}
function Empty({ className, children, ...props }: EmptyRootProps) {
  return (
    <div
      className={cn(
        "flex flex-col p-8 text-center h-full w-full items-center justify-center",
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
}
Empty.displayName = "Empty";

type EmptyIconProps = React.HTMLAttributes<HTMLDivElement> & {
  children?: React.ReactNode;
};

Empty.Icon = function EmptyIcon({ className, children }: EmptyIconProps) {
  return (
    <div className={cn("flex h-28 w-28 justify-center items-center", className)}>
      <div className="relative z-10">
        <div className="absolute top-0 left-1/2 -translate-x-1/2 bg-gradient-to-r from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-32 h-[1px] " />
        <div className="absolute bottom-0 left-1/2 -translate-x-1/2 bg-gradient-to-r from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-32 h-[1px]  " />
        <div className="absolute left-0 top-1/2 -translate-y-1/2 h-32 bg-gradient-to-t from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-[1px]" />
        <div className="absolute right-0 top-1/2 -translate-y-1/2 h-32 bg-gradient-to-t from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-[1px] " />
        <div className="flex w-16 h-16 items-center justify-center rounded-2xl bg-gray-2 border-[1px] border-[hsla(240,100%,10%,0.06)] dark:border-[hsla(211,66%,92%,0.2)] text-accent-12 z-50 [&_svg]:pointer-events-none [&_svg]:size-7 [&_svg]:shrink-0">
          <div>{children || <Ufo iconSize="2xl-regular" />}</div>
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

type EmptyActionsProps = React.HTMLAttributes<HTMLDivElement>;

Empty.Actions = function EmptyActions({ className, children, ...props }: EmptyActionsProps) {
  return (
    <div className={cn("w-full flex items-center justify-center gap-4 mt-2", className)} {...props}>
      {children}
    </div>
  );
};

export { Empty };
