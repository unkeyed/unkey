import * as React from "react";
import { cn } from "../../lib/utils";

const PageHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "mx-auto flex w-full max-w-7xl items-start justify-between gap-3 px-4 pt-6",
        "group-data-[width=full]/page:max-w-none group-data-[width=full]/page:border-b group-data-[width=full]/page:border-grayA-4 group-data-[width=full]/page:pb-4 group-data-[width=full]/page:pt-4",
        className,
      )}
      {...props}
    />
  ),
);
PageHeader.displayName = "PageHeader";

const PageHeaderContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex flex-col gap-0.5 min-w-0", className)} {...props} />
  ),
);
PageHeaderContent.displayName = "PageHeaderContent";

const PageHeaderTitle = React.forwardRef<
  HTMLHeadingElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
  <h1
    ref={ref}
    className={cn(
      "text-[22px] font-semibold tracking-tight leading-tight text-accent-12 m-0",
      className,
    )}
    {...props}
  />
));
PageHeaderTitle.displayName = "PageHeaderTitle";

const PageHeaderDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
  <p ref={ref} className={cn("text-[13px] leading-5 text-accent-11 m-0", className)} {...props} />
));
PageHeaderDescription.displayName = "PageHeaderDescription";

const PageHeaderActions = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex items-center gap-2 shrink-0", className)} {...props} />
  ),
);
PageHeaderActions.displayName = "PageHeaderActions";

export { PageHeader, PageHeaderContent, PageHeaderTitle, PageHeaderDescription, PageHeaderActions };
