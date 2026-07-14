import type * as React from "react";
import { cn } from "../../lib/utils";

function ResourceList({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex w-full flex-col gap-3", className)} {...props} />;
}

function ResourceListHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div className={cn("flex flex-col items-stretch gap-2 md:flex-row", className)} {...props} />
  );
}

function ResourceListContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div className={cn("overflow-hidden rounded-lg border border-grayA-4", className)} {...props} />
  );
}

function ResourceListFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("flex items-center justify-end border-grayA-4 border-t px-4 py-3", className)}
      {...props}
    />
  );
}

export { ResourceList, ResourceListContent, ResourceListFooter, ResourceListHeader };
