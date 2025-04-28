import {
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar";
import { useDelayLoader } from "@/hooks/use-delay-loader";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent } from "@radix-ui/react-collapsible";
import { CaretRight } from "@unkey/icons";
import { usePathname, useRouter } from "next/navigation";
import { useLayoutEffect, useState, useTransition } from "react";
import type { NavItem } from "../../../workspace-navigations";
import { NavLink } from "../nav-link";
import { AnimatedLoadingSpinner } from "./animated-loading-spinner";
import { getButtonStyles } from "./utils";

const paths = ["/apis", "/ratelimits"];

export const NestedNavItem = ({
  item,
  onLoadMore,
  depth = 0,
  maxDepth = 1,
  isSubItem = false,
}: {
  item: NavItem;
  onLoadMore?: () => void;
  depth?: number;
  maxDepth?: number;
  isSubItem?: boolean;
}) => {
  const [parentIsPending, startParentTransition] = useTransition();
  const showParentLoader = useDelayLoader(parentIsPending);
  const router = useRouter();
  const pathname = usePathname();
  const [subPending, setSubPending] = useState<Record<string, boolean>>({});
  const [isOpen, setIsOpen] = useState(false);
  const [isChildrenOpen, setIsChildrenOpen] = useState(false);

  const Icon = item.icon;
  const hasChildren = item.items && item.items.length > 0;

  useLayoutEffect(() => {
    if (!hasChildren || !pathname) {
      return;
    }

    const hasMatchingChild = item.items?.some(
      (subItem) =>
        subItem.href === pathname || subItem.items?.some((child) => child.href === pathname),
    );

    setIsChildrenOpen(!!hasMatchingChild);

    if (paths.includes(pathname) && pathname.startsWith(item.href)) {
      setIsOpen(true);
    }
  }, [pathname, item.items, item.href, hasChildren]);

  const handleMenuItemClick = (e: React.MouseEvent) => {
    // If the item has children, toggle the open state
    if (hasChildren && !isSubItem) {
      e.preventDefault();
      setIsOpen((prev) => !prev);
      if (isOpen) {
        return;
      }
    }

    // If the item has a href, navigate to it
    if (item.href) {
      if (!item.external) {
        // Show loading state ONLY for parent
        startParentTransition(() => {
          router.push(item.href);
        });
      } else {
        // For external links, open in new tab
        window.open(item.href, "_blank");
      }
    }
  };

  // Render a sub-item, potentially recursively if it has children
  const renderSubItem = (subItem: NavItem, index: number) => {
    const SubIcon = subItem.icon;
    const isLoadMoreButton = subItem.loadMoreAction === true;
    const hasChildren = subItem.items && subItem.items.length > 0;

    // If this subitem has children and is not at max depth, render it as another NestedNavItem
    if (hasChildren && depth < maxDepth) {
      return (
        <SidebarMenuSubItem key={subItem.label ?? index}>
          <NestedNavItem
            item={subItem}
            onLoadMore={onLoadMore}
            depth={depth + 1}
            maxDepth={maxDepth}
            isSubItem={true}
          />
        </SidebarMenuSubItem>
      );
    }

    // Otherwise render as a regular sub-item
    const handleSubItemClick = () => {
      if (isLoadMoreButton && onLoadMore) {
        onLoadMore();
        return;
      }

      if (!subItem.external && subItem.href) {
        // Track loading state for this specific sub-item
        const updatedPending = { ...subPending };
        updatedPending[subItem.label as string] = true;
        setSubPending(updatedPending);

        // Use a separate transition for sub-items
        // This prevents parent from showing loader
        const subItemTransition = () => {
          router.push(subItem.href);
          // Reset loading state after transition
          setTimeout(() => {
            const resetPending = { ...subPending };
            resetPending[subItem.label as string] = false;
            setSubPending(resetPending);
          }, 300);
        };

        // Execute transition without affecting parent's isPending state
        subItemTransition();
      }
    };

    return (
      <SidebarMenuSubItem key={subItem.label ?? index}>
        <NavLink
          href={subItem.href}
          external={subItem.external}
          onClick={handleSubItemClick}
          isLoadMoreButton={isLoadMoreButton}
        >
          <SidebarMenuButton
            isActive={subItem.active}
            className={getButtonStyles(subItem.active, subPending[subItem.label as string])}
          >
            {SubIcon ? (
              subPending[subItem.label as string] ? (
                <AnimatedLoadingSpinner />
              ) : (
                <SubIcon size="xl-medium" />
              )
            ) : null}
            <span className="truncate">{subItem.label}</span>
            {subItem.tag && <div className="ml-auto">{subItem.tag}</div>}
          </SidebarMenuButton>
        </NavLink>
      </SidebarMenuSubItem>
    );
  };

  return (
    <Collapsible
      asChild
      open={isSubItem ? isChildrenOpen : isOpen}
      onOpenChange={isSubItem ? setIsChildrenOpen : setIsOpen}
      className="group/collapsible"
    >
      <SidebarMenuItem>
        <SidebarMenuButton
          tooltip={item.tooltip}
          // Only highlight if this item itself is active, not if its children are active
          isActive={item.active}
          className={cn(
            // Only highlight if this item itself is active, not if its children are active
            getButtonStyles(item.active, showParentLoader),
            "cursor-pointer relative",
          )}
          onClick={handleMenuItemClick}
        >
          {Icon ? showParentLoader ? <AnimatedLoadingSpinner /> : <Icon size="xl-medium" /> : null}
          <span className="truncate max-w-[180px]">{item.label}</span>
          {item.tag && <div className="ml-auto mr-2">{item.tag}</div>}
          {/* Chevron icon to indicate there are children */}
          {item.items && item.items.length > 0 && (
            <div className="w-5 h-5 flex items-center justify-center flex-shrink-0">
              <CaretRight
                className={cn(
                  "transition-transform duration-200 text-gray-9 !w-[9px] !h-[9px]",
                  isOpen ? "rotate-90" : "rotate-0",
                )}
                size="sm-bold"
              />
            </div>
          )}
        </SidebarMenuButton>
        {item.items && item.items.length > 0 && (
          <CollapsibleContent>
            <SidebarMenuSub depth={depth} maxDepth={maxDepth}>
              {item.items.map((subItem, index) => renderSubItem(subItem, index))}
            </SidebarMenuSub>
          </CollapsibleContent>
        )}
      </SidebarMenuItem>
    </Collapsible>
  );
};
