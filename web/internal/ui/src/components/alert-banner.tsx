import type * as React from "react";
import { type VariantProps, cva } from "../lib/cva";
import { cn } from "../lib/utils";

const alertBannerVariants = cva(
  "group/alert-banner grid w-full border [&>svg]:col-start-1 [&>svg]:row-start-1 [&>svg]:shrink-0",
  {
    variants: {
      appearance: {
        banner: [
          "grid-cols-[auto_1fr_auto] items-center rounded-lg px-4 py-3 [&>svg]:mr-3",
          "has-[>[data-slot=alert-banner-title]+[data-slot=alert-banner-description]]:items-start",
          "has-[>[data-slot=alert-banner-title]+[data-slot=alert-banner-description]]:p-4",
          "has-[>[data-slot=alert-banner-title]+[data-slot=alert-banner-description]]:[&>svg]:translate-y-0.5",
          "has-[>[data-slot=alert-banner-title]+[data-slot=alert-banner-description]]:*:data-[slot=alert-banner-actions]:row-end-3",
        ],
        card: [
          "grid-cols-[auto_auto_1fr_auto] items-center gap-x-2 rounded-md bg-raised px-3 pt-2.5 text-gray-12",
          "not-has-[>[data-slot=alert-banner-description]]:pb-2.5",
        ],
      },
      variant: {
        default: "",
        error: "",
        warning: "",
        success: "",
        info: "",
      },
    },
    compoundVariants: [
      {
        appearance: "banner",
        variant: "default",
        className: "bg-raised [&>svg]:text-gray-12 *:data-[slot=alert-banner-title]:text-gray-12",
      },
      {
        appearance: "banner",
        variant: "error",
        className:
          "border-errorA-4 bg-errorA-2 [&>svg]:text-error-11 *:data-[slot=alert-banner-title]:text-error-11",
      },
      {
        appearance: "banner",
        variant: "warning",
        className:
          "border-warningA-6 bg-warningA-2 [&>svg]:text-warning-11 *:data-[slot=alert-banner-title]:text-warning-11",
      },
      {
        appearance: "banner",
        variant: "success",
        className:
          "border-successA-6 bg-successA-2 [&>svg]:text-success-11 *:data-[slot=alert-banner-title]:text-success-11",
      },
      {
        appearance: "banner",
        variant: "info",
        className:
          "border-infoA-4 bg-infoA-2 [&>svg]:text-info-11 *:data-[slot=alert-banner-title]:text-info-11",
      },
      // A one-color gradient sets background-image, which tints over the card's
      // bg-raised instead of replacing it as a bg-* color would
      {
        appearance: "card",
        variant: "warning",
        className:
          "border-warningA-6 bg-linear-to-b from-warningA-2/50 to-warningA-2/50 [&>svg]:text-warning-9",
      },
      {
        appearance: "card",
        variant: "error",
        className:
          "border-errorA-6 bg-linear-to-b from-errorA-2/50 to-errorA-2/50 [&>svg]:text-error-9",
      },
    ],
    defaultVariants: {
      variant: "default",
      appearance: "banner",
    },
  },
);

type BannerVariant = NonNullable<VariantProps<typeof alertBannerVariants>["variant"]>;

type AlertBannerProps = React.ComponentProps<"div"> &
  (
    | { appearance?: "banner"; variant?: BannerVariant }
    | { appearance: "card"; variant: Extract<BannerVariant, "warning" | "error"> }
  );

function AlertBanner({ className, variant, appearance = "banner", ...props }: AlertBannerProps) {
  return (
    <div
      data-slot="alert-banner"
      data-appearance={appearance}
      role={appearance === "card" ? "status" : "alert"}
      className={cn(alertBannerVariants({ variant, appearance }), className)}
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
        "col-start-2 min-w-0 text-sm leading-5 text-gray-11",
        "group-data-[appearance=banner]/alert-banner:[[data-slot=alert-banner-title]+&]:mt-0.5",
        "group-data-[appearance=card]/alert-banner:col-span-full group-data-[appearance=card]/alert-banner:col-start-1",
        "group-data-[appearance=card]/alert-banner:-mx-3 group-data-[appearance=card]/alert-banner:mt-2.5",
        "group-data-[appearance=card]/alert-banner:border-t group-data-[appearance=card]/alert-banner:border-grayA-4",
        "group-data-[appearance=card]/alert-banner:px-3 group-data-[appearance=card]/alert-banner:py-2.5",
        linkStyles,
        className,
      )}
      {...props}
    />
  );
}

function AlertBannerMeta({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-banner-meta"
      className={cn(
        "col-start-2 min-w-0 truncate text-xs leading-5 text-gray-10",
        "group-data-[appearance=card]/alert-banner:col-start-3",
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
        "group-data-[appearance=card]/alert-banner:col-start-4 group-data-[appearance=card]/alert-banner:ml-0",
        className,
      )}
      {...props}
    />
  );
}

export {
  AlertBanner,
  AlertBannerTitle,
  AlertBannerDescription,
  AlertBannerMeta,
  AlertBannerActions,
};
