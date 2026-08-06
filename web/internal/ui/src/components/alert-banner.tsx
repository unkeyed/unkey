import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "../lib/utils";

const alertBannerVariants = cva(
  [
    "grid w-full grid-cols-[auto_1fr_auto] items-start rounded-lg border px-4 py-3",
    "has-[>[data-slot=alert-banner-title]]:p-4",
    "[&>svg]:col-start-1 [&>svg]:row-start-1 [&>svg]:mr-3 [&>svg]:shrink-0 [&>svg]:translate-y-0.5",
    "has-[>[data-slot=alert-banner-title]]:has-[>[data-slot=alert-banner-description]]:*:data-[slot=alert-banner-actions]:row-end-3",
  ],
  {
    variants: {
      variant: {
        default:
          "border-grayA-4 bg-grayA-2 [&>svg]:text-gray-12 *:data-[slot=alert-banner-title]:text-gray-12",
        error:
          "border-errorA-4 bg-errorA-2 [&>svg]:text-error-11 *:data-[slot=alert-banner-title]:text-error-11",
        warning:
          "border-warningA-6 bg-warningA-2 [&>svg]:text-warning-11 *:data-[slot=alert-banner-title]:text-warning-11",
        success:
          "border-successA-6 bg-successA-2 [&>svg]:text-success-11 *:data-[slot=alert-banner-title]:text-success-11",
        info: "border-infoA-4 bg-infoA-2 [&>svg]:text-info-11 *:data-[slot=alert-banner-title]:text-info-11",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

function AlertBanner({
  className,
  variant,
  ...props
}: React.ComponentProps<"div"> & VariantProps<typeof alertBannerVariants>) {
  return (
    <div
      data-slot="alert-banner"
      role="alert"
      className={cn(alertBannerVariants({ variant }), className)}
      {...props}
    />
  );
}

const linkStyles = "[&_a]:underline [&_a]:underline-offset-2 [&_a]:hover:text-gray-12";

function AlertBannerTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-banner-title"
      className={cn("col-start-2 min-w-0 text-sm font-medium leading-5", linkStyles, className)}
      {...props}
    />
  );
}

function AlertBannerDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-banner-description"
      className={cn(
        "col-start-2 min-w-0 text-[13px] leading-5 text-gray-11",
        "[[data-slot=alert-banner-title]+&]:mt-0.5",
        linkStyles,
        className,
      )}
      {...props}
    />
  );
}

function AlertBannerActions({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-banner-actions"
      className={cn(
        "col-start-3 row-start-1 ml-3 flex shrink-0 items-center gap-2 self-center",
        className,
      )}
      {...props}
    />
  );
}

export { AlertBanner, AlertBannerTitle, AlertBannerDescription, AlertBannerActions };
