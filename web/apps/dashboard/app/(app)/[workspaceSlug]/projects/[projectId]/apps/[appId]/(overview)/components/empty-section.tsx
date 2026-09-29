"use client";

import { IconCubeOutline18 } from "@unkey/icons";
import {
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";
import { cn } from "cn";
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
  icon = <IconCubeOutline18 />,
  className,
}: EmptySectionProps) => (
  <EmptyState className={cn("min-h-[150px]", className)}>
    <EmptyStateIcon>{icon}</EmptyStateIcon>
    <EmptyStateHeader>
      <EmptyStateTitle>{title}</EmptyStateTitle>
      <EmptyStateDescription>{description}</EmptyStateDescription>
    </EmptyStateHeader>
    {children && <EmptyStateActions>{children}</EmptyStateActions>}
  </EmptyState>
);
