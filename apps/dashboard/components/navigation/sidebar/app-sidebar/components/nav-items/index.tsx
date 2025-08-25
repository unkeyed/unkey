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
};

export const NavItems = ({ item, onLoadMore, onToggleCollapse }: NavProps) => {
  if (!item.items || item.items.length === 0) {
    return <FlatNavItem item={item} onLoadMore={onLoadMore} />;
  }
  return <NestedNavItem item={item} onLoadMore={onLoadMore} onToggleCollapse={onToggleCollapse} />;
};
