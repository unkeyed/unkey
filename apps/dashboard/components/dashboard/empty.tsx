import type * as React from "react";

import { cn } from "@/lib/utils";

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
      <div className="flex flex-col items-center justify-center h-full w-full">{children}</div>
    </div>
  );
}

// Empty.Icon = function EmptyIcon({ children }: { children: React.ReactNode }) {
//     return (
//         <div className="flex items-center justify-center w-20 h-20 rounded-2xl bg-background-subtle border border-gray-100">
//             {children}
//         </div>
//     );
// };
Empty.Icon = function EmptyIcon({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-28 w-28 justify-center items-center">
      <div className="relative z-10">
        {/* Add gradient to lines */}
        <div className="absolute top-[1-px] left-1/2 -translate-x-1/2 border-t-[1px] border-border w-32 z-10" />
        <div className="absolute bottom-[-1px] left-1/2 -translate-x-1/2 border-t-[1px] border-border w-32 z-10" />
        <div className="absolute left-[-1px] top-1/2 -translate-y-1/2 h-32 border-l-[1px] border-border z-10" />
        <div className="absolute right-[-1px] top-1/2 -translate-y-1/2 h-32 border-l-[1px] border-border z-10" />
        <div className="relative flex w-16 h-16 items-center justify-center rounded-2xl bg-background-subtle border border-border text-content z-50">
          {children}
        </div>
      </div>
    </div>
  );
};

type EmptyTitleProps = React.HTMLAttributes<HTMLHeadingElement>;

Empty.Title = function EmptyTitle({ className, ...props }: EmptyTitleProps) {
  return <h2 className={cn("mt-6 text-base font-semibold text-zinc-900", className)} {...props} />;
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
      className={cn("flex gap-4 mt-4 px-3 items-center justify-center wrap ", className, {
        ...props,
      })}
    >
      {children}
    </div>
  );
};
