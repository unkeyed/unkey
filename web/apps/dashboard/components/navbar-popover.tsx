"use client";
import React, { type PropsWithChildren } from "react";

export type QuickNavItem = {
  id: string;
  label: React.ReactNode;
  shortcut?: string;
  href?: string;
  onClick?: () => void;
  className?: string;
  itemClassName?: string;
  hideRightIcon?: boolean;
  disabled?: boolean;
  disabledTooltip?: string;
};

type QuickNavPopoverProps = {
  items: QuickNavItem[];
  title?: string;
  shortcutKey?: string;
  onItemSelect?: (item: QuickNavItem) => void;
  /**
   * Explicitly specify which item should be highlighted as active.
   * Takes precedence over path-based matching.
   */
  activeItemId?: string;
  /**
   * Threshold for when to use virtualization.
   * Lists with fewer items than this will render without virtualization.
   * @default 10
   */
  virtualizationThreshold?: number;
};

export const QuickNavPopover = (props: PropsWithChildren<QuickNavPopoverProps>) => {
  return <QuickNavLabel>{props.children}</QuickNavLabel>;
};

const QuickNavLabel = ({ children }: PropsWithChildren) => {
  if (React.isValidElement<{ children?: React.ReactNode }>(children)) {
    const inner = React.Children.toArray(children.props.children);
    return <>{inner[0] ?? null}</>;
  }
  return <>{children}</>;
};
