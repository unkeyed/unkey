"use client";

import { cn } from "@/lib/utils";
import { Empty } from "@unkey/ui";
import { IconLink4Outline18 } from "nucleo-ui-outline-18";
import type { PropsWithChildren, ReactNode } from "react";

type EmptySectionProps = PropsWithChildren<{
  title: string;
  description: string;
  icon?: ReactNode;
  className?: string;
}>;

export const EmptySection = ({
  title,
  description,
  children,
  icon = <IconLink4Outline18 className="size-6" />,
  className,
}: EmptySectionProps) => (
  <Empty
    className={cn(
      "min-h-[150px] rounded-lg border border-dashed border-gray-4 bg-gray-1/50",
      className,
    )}
  >
    <Empty.Icon>{icon}</Empty.Icon>
    <Empty.Title>{title}</Empty.Title>
    <Empty.Description className="max-w-sm">{description}</Empty.Description>
    {children && <Empty.Actions>{children}</Empty.Actions>}
  </Empty>
);
