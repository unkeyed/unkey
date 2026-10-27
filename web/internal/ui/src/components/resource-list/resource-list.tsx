import type * as React from "react";
import { cn } from "../../lib/utils";
import { Button } from "../buttons/button";

type ResourceListFooterVariant = "framed" | "bare";

const FOOTER_VARIANTS: Record<ResourceListFooterVariant, string> = {
  framed: "border-t px-4 py-3",
  bare: "",
};

function ResourceList({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex w-full flex-col gap-3", className)} {...props} />;
}

function ResourceListHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("flex flex-col items-stretch gap-2 md:flex-row md:items-center", className)}
      {...props}
    />
  );
}

function ResourceListSearch({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex h-8 w-full items-center md:w-80", className)} {...props} />;
}

function ResourceListContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div className={cn("overflow-hidden rounded-lg border bg-raised", className)} {...props} />
  );
}

function ResourceListBody({ className, ...props }: React.ComponentProps<"ul">) {
  return <ul className={cn("divide-y divide-grayA-4", className)} {...props} />;
}

function ResourceListItem({ className, ...props }: React.ComponentProps<"li">) {
  return <li className={cn("relative", className)} {...props} />;
}

/**
 * The whole row activates through a button stretched under its cells.
 * Give interactive cells `relative` so they stay clickable above it.
 */
function ResourceListRow({
  label,
  onActivate,
  expanded,
  className,
  children,
  ...props
}: React.ComponentProps<"div"> & {
  label: string;
  onActivate: () => void;
  expanded?: boolean;
}) {
  return (
    <div
      className={cn(
        "relative h-12 px-4 transition-colors hover:bg-grayA-2 has-[>button:focus-visible]:bg-grayA-2 has-[>button[aria-expanded=true]]:bg-grayA-2",
        className,
      )}
      {...props}
    >
      <button
        type="button"
        aria-label={label}
        aria-expanded={expanded}
        onClick={onActivate}
        className="absolute inset-0 cursor-pointer outline-hidden"
      />
      {children}
    </div>
  );
}

function ResourceListFooter({
  variant = "framed",
  className,
  ...props
}: React.ComponentProps<"div"> & { variant?: ResourceListFooterVariant }) {
  return (
    <div
      className={cn("flex items-center justify-end", FOOTER_VARIANTS[variant], className)}
      {...props}
    />
  );
}

function ResourceListShowMore({
  loading = false,
  onClick,
}: {
  loading?: boolean;
  onClick?: () => void;
}) {
  return (
    <Button
      size="md"
      variant="outline"
      className="w-full"
      loading={loading}
      disabled={loading}
      onClick={onClick}
    >
      Show more
    </Button>
  );
}

export {
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListFooter,
  ResourceListHeader,
  ResourceListItem,
  ResourceListRow,
  ResourceListSearch,
  ResourceListShowMore,
};
export type { ResourceListFooterVariant };
