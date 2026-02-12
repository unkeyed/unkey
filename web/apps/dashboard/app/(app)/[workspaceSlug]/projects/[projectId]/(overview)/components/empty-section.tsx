"use client";

import { cn } from "@/lib/utils";
import { Link4 } from "@unkey/icons";
import { Empty } from "@unkey/ui";
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
  icon = <Link4 className="size-6" />,
  className,
}: EmptySectionProps) => (
  <Empty
    className={cn(
      "min-h-[150px] rounded-[14px] border border-dashed border-gray-4 bg-gray-1/50",
      className,
    )}
  >
    <Empty.Icon>{icon}</Empty.Icon>
    <Empty.Title>{title}</Empty.Title>
    <Empty.Description className="max-w-sm">{description}</Empty.Description>
    {children && <Empty.Actions>{children}</Empty.Actions>}
  </Empty>
);
