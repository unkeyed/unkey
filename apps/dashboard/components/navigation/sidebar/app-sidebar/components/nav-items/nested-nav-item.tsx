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
import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { NavItem } from "../../../workspace-navigations";
import { NavLink } from "../nav-link";
import { AnimatedLoadingSpinner } from "./animated-loading-spinner";
import { getButtonStyles } from "./utils";

export const NestedNavItem = ({
  item,
  onLoadMore,
  depth = 0,
  maxDepth = 1,
}: {
  item: NavItem;
  onLoadMore?: () => void;
  depth?: number;
  maxDepth?: number;
}) => {
  const [parentIsPending, startParentTransition] = useTransition();
  const showParentLoader = useDelayLoader(parentIsPending);
  const router = useRouter();
  const Icon = item.icon;
  const [subPending, setSubPending] = useState<Record<string, boolean>>({});
  const [isOpen, setIsOpen] = useState(item.active);

  const handleChevronClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    setIsOpen((prev) => !prev);
  };

  const handleMenuItemClick = () => {
    if (item.href) {
      if (!item.external) {
        // Show loading state ONLY for parent
        startParentTransition(() => {
          router.push(item.href);
        });
      } else {
        // For external links, let the NavLink handle it
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
            {subPending[subItem.label as string] ? (
              <AnimatedLoadingSpinner />
            ) : SubIcon ? (
              <SubIcon />
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
      open={isOpen}
      onOpenChange={setIsOpen}
      defaultOpen={item.active}
      className="group/collapsible"
    >
      <SidebarMenuItem>
        <SidebarMenuButton
          tooltip={item.tooltip}
          isActive={item.active}
          className={cn(getButtonStyles(item.active, showParentLoader), "cursor-pointer relative")}
          onClick={handleMenuItemClick}
        >
          {showParentLoader ? <AnimatedLoadingSpinner /> : Icon ? <Icon /> : null}
          <span className="truncate max-w-[180px]">{item.label}</span>
          {item.tag && <div className="ml-auto mr-2">{item.tag}</div>}
          {/* Embed the chevron inside the button with its own click handler */}
          {item.items && item.items.length > 0 && (
            // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
            <div
              className="w-5 h-5 flex items-center justify-center flex-shrink-0"
              onClick={(e) => handleChevronClick(e)}
            >
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
