import type * as React from "react";
import { cn } from "../../lib/utils";

function PageHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn(
        // Center the title against the actions when there's no description (a
        // single line), but top-align once a description (<p>) is present.
        "mx-auto flex w-full max-w-7xl items-center justify-between gap-3 px-6 pt-6 has-[p]:items-start",
        // More breathing room above the fold on large desktops (not laptops).
        "2xl:pt-8",
        "group-data-[width=full]/page:max-w-none group-data-[width=full]/page:border-b group-data-[width=full]/page:border-grayA-4 group-data-[width=full]/page:pb-4 group-data-[width=full]/page:pt-4 group-data-[width=full]/page:2xl:pt-4",
        className,
      )}
      {...props}
    />
  );
}

function PageHeaderContent({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex flex-col gap-0.5 min-w-0", className)} {...props} />;
}

function PageHeaderTitle({ className, ...props }: React.ComponentProps<"h1">) {
  return (
    <h1
      className={cn(
        "text-[22px] font-semibold tracking-tight leading-tight text-accent-12 m-0",
        className,
      )}
      {...props}
    />
  );
}

function PageHeaderDescription({ className, ...props }: React.ComponentProps<"p">) {
  return <p className={cn("text-[13px] leading-5 text-accent-11 m-0", className)} {...props} />;
}

function PageHeaderActions({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex items-center gap-2 shrink-0", className)} {...props} />;
}

export { PageHeader, PageHeaderContent, PageHeaderTitle, PageHeaderDescription, PageHeaderActions };
