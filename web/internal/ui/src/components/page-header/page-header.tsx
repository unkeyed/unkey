import * as React from "react";
import { cn } from "../../lib/utils";

/**
 * The page header sits at the top of a page's content area, below the global
 * chrome. Compose it from its parts rather than passing a bag of props:
 *
 * ```tsx
 * <PageHeader>
 *   <PageHeaderContent>
 *     <PageHeaderTitle>Settings</PageHeaderTitle>
 *     <PageHeaderDescription>Manage your workspace</PageHeaderDescription>
 *   </PageHeaderContent>
 *   <PageHeaderActions>
 *     <Button>Save</Button>
 *   </PageHeaderActions>
 * </PageHeader>
 * ```
 *
 * For a status badge beside the title, wrap the title in a row:
 * `<div className="flex items-center gap-2"><PageHeaderTitle>…</PageHeaderTitle><Badge/></div>`.
 */
const PageHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex items-start justify-between gap-3", className)} {...props} />
  ),
);
PageHeader.displayName = "PageHeader";

/** The left column of a {@link PageHeader}: stacks the title and description. */
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

/** The right-aligned action cluster of a {@link PageHeader}. */
const PageHeaderActions = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex items-center gap-2 shrink-0", className)} {...props} />
  ),
);
PageHeaderActions.displayName = "PageHeaderActions";

export { PageHeader, PageHeaderContent, PageHeaderTitle, PageHeaderDescription, PageHeaderActions };
