import type { NavItem } from "../../../workspace-navigations";
import { FlatNavItem } from "./flat-nav-item";
import { NestedNavItem } from "./nested-nav-item";

export const NavItems = ({
  item,
  onLoadMore,
}: {
  item: NavItem & {
    items?: (NavItem & { loadMoreAction?: boolean })[];
    loadMoreAction?: boolean;
  };
  onLoadMore?: () => void;
}) => {
  if (!item.items || item.items.length === 0) {
    return <FlatNavItem item={item} onLoadMore={onLoadMore} />;
  }
  return <NestedNavItem item={item} onLoadMore={onLoadMore} />;
};
