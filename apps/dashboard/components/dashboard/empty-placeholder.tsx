import type * as React from "react";

import { cn } from "@/lib/utils";

export function EmptyPlaceholder({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "animate-in relative fade-in-50 flex min-h-[400px] flex-col items-center justify-center rounded-md border border-dashed p-8 text-center",
        className,
      )}
      {...props}
    >
      <div className="mx-auto flex max-w-[420px] flex-col items-center justify-center text-center">
        {children}
      </div>
    </div>
  );
}

EmptyPlaceholder.Icon = function EmptyPlaceHolderIcon({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-center w-20 h-20 rounded-full bg-background-subtle">
      {children}
    </div>
  );
};

type EmptyPlaceholderTitleProps = React.HTMLAttributes<HTMLHeadingElement>;

EmptyPlaceholder.Title = function EmptyPlaceholderTitle({
  className,
  ...props
}: EmptyPlaceholderTitleProps) {
  return <h2 className={cn("mt-6 text-xl font-semibold", className)} {...props} />;
};

type EmptyPlaceholderDescriptionProps = React.HTMLAttributes<HTMLParagraphElement>;

EmptyPlaceholder.Description = function EmptyPlaceholderDescription({
  className,
  ...props
}: EmptyPlaceholderDescriptionProps) {
  return (
    <p
      className={cn(
        "text-content-subtle mb-8 mt-2 text-center text-sm font-normal leading-6",
        className,
      )}
      {...props}
    />
  );
};
