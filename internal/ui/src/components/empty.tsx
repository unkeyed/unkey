import type * as React from "react";

import { cn } from "../lib/utils";

interface EmptyRootProps extends React.HTMLAttributes<HTMLDivElement> {
  fill: boolean | undefined;
}
export function Empty({ className, children, ...props }: EmptyRootProps) {
  return (
    <div
      className={cn(
        props.fill ? "h-full w-full" : "",
        "animate-in fade-in-50 flex-col p-8 text-center justify-center",
        className,
      )}
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
        {/* Add gradient to lines */}
        <div className="absolute top-[1-px] left-1/2 -translate-x-1/2  bg-gradient-to-r from-transparent via-gray-500 to-transparent w-32 h-[1px] z-10" />
        <div className="absolute bottom-[-1px] left-1/2 -translate-x-1/2 bg-gradient-to-r from-transparent via-gray-500 to-transparent w-32 h-[1px] z-10 " />
        <div className="absolute left-[-1px] top-1/2 -translate-y-1/2 h-32 bg-gradient-to-t from-transparent via-gray-500 to-transparent w-[1px] z-10" />
        <div className="absolute right-[-1px] top-1/2 -translate-y-1/2 h-32 bg-gradient-to-t from-transparent via-gray-500 to-transparent w-[1px] z-10" />
        <div className="relative flex w-16 h-16 items-center justify-center rounded-2xl bg-gray-3 border border-gray-500 text-accent-12 z-50">
          {children}
        </div>
      </div>
    </div>
  );
};

type EmptyTitleProps = React.HTMLAttributes<HTMLHeadingElement>;

Empty.Title = function EmptyTitle({ className, ...props }: EmptyTitleProps) {
  return <h2 className={cn("mt-4 text-base font-semibold text-content ", className)} {...props} />;
};

type EmptyDescriptionProps = React.HTMLAttributes<HTMLParagraphElement>;

Empty.Description = function EmptyDescription({ className, ...props }: EmptyDescriptionProps) {
  return (
    <p
      className={cn(
        "text-content-subtle text-center text-sm font-normal leading-6 mt-2",
        className,
      )}
      {...props}
    />
  );
};

type EmptyActionProps = React.HTMLAttributes<HTMLDivElement>;

Empty.Action = function EmptyAction({ className, children, ...props }: EmptyActionProps) {
  return (
    <div
      className={cn("flex gap-4 mt-2 px-3 items-center justify-center leading-6 mt-4", className, {
        ...props,
      })}
    >
      {children}
    </div>
  );
};
