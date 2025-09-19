"use client";
import type { NavItem } from "../../../workspace-navigations";
import { FlatNavItem } from "./flat-nav-item";
import { NestedNavItem } from "./nested-nav-item";

export type NavProps = {
  item: NavItem & {
    items?: (NavItem & { loadMoreAction?: boolean })[];
    loadMoreAction?: boolean;
  };
  onLoadMore?: (item: NavItem) => void;
  onToggleCollapse?: (item: NavItem, isOpen: boolean) => void;
  forceCollapsed?: boolean;
  className?: string;
};

export const NavItems = ({
  item,
  forceCollapsed,
  onLoadMore,
  onToggleCollapse,
  className,
}: NavProps) => {
  if (!item.items || item.items.length === 0) {
    return <FlatNavItem item={item} onLoadMore={onLoadMore} className={className} />;
  }
  return (
    <NestedNavItem
      item={item}
      onLoadMore={onLoadMore}
      forceCollapsed={forceCollapsed}
      onToggleCollapse={onToggleCollapse}
      className={className}
    />
  );
};
