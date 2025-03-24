import {
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar";
import { useDelayLoader } from "@/hooks/use-delay-loader";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@radix-ui/react-collapsible";
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
}: {
  item: NavItem;
  onLoadMore?: () => void;
}) => {
  const [isPending, startTransition] = useTransition();
  const showLoader = useDelayLoader(isPending);
  const router = useRouter();
  const Icon = item.icon;
  // For loading indicators in sub-items
  const [subPending, setSubPending] = useState<Record<string, boolean>>({});

  // Initialize with the prop value, defaulting to true if not specified
  const [shouldShowSubItems, setShouldShowSubItems] = useState(item.showSubItems !== false);

  // To manage the collapsible state separately from the visibility toggle
  const [isOpen, setIsOpen] = useState(item.active);

  // Handler for the caret button click
  const handleToggleVisibility = (e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent triggering the collapsible
    setShouldShowSubItems((prev) => !prev);
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
        <CollapsibleTrigger asChild>
          <SidebarMenuButton
            tooltip={item.tooltip}
            isActive={item.active}
            className={getButtonStyles(item.active, showLoader)}
          >
            {showLoader ? <AnimatedLoadingSpinner /> : <Icon />}
            <span>{item.label}</span>
            {item.tag && <div className="ml-auto mr-2">{item.tag}</div>}
            {/* Only show the caret if there are subitems */}
            {item.items && item.items.length > 0 && (
              <button className="w-5 h-5" type="button" onClick={handleToggleVisibility}>
                <CaretRight
                  className={cn(
                    "transition-transform duration-200 text-gray-9 !w-3 !h-3",
                    shouldShowSubItems && isOpen ? "rotate-90" : "rotate-0",
                  )}
                  size="sm-bold"
                />
              </button>
            )}
          </SidebarMenuButton>
        </CollapsibleTrigger>

        {/* Only render CollapsibleContent if subitems should be shown */}
        {shouldShowSubItems && item.items && item.items.length > 0 && (
          <CollapsibleContent>
            <SidebarMenuSub>
              {item.items.map((subItem) => {
                const SubIcon = subItem.icon;
                const isLoadMoreButton = subItem.loadMoreAction === true;
                const handleSubItemClick = () => {
                  if (isLoadMoreButton && onLoadMore) {
                    onLoadMore();
                    return;
                  }
                  if (!subItem.external) {
                    // Track loading state for this specific sub-item
                    const updatedPending = { ...subPending };
                    updatedPending[subItem.label] = true;
                    setSubPending(updatedPending);
                    startTransition(() => {
                      router.push(subItem.href);
                      // Reset loading state after transition
                      setTimeout(() => {
                        const resetPending = { ...subPending };
                        resetPending[subItem.label] = false;
                        setSubPending(resetPending);
                      }, 300);
                    });
                  }
                };
                return (
                  <SidebarMenuSubItem key={subItem.label}>
                    <NavLink
                      href={subItem.href}
                      external={subItem.external}
                      onClick={handleSubItemClick}
                      isLoadMoreButton={isLoadMoreButton}
                    >
                      <SidebarMenuSubButton
                        isActive={subItem.active}
                        className={getButtonStyles(subItem.active, subPending[subItem.label])}
                      >
                        {subPending[subItem.label] ? (
                          <AnimatedLoadingSpinner />
                        ) : SubIcon ? (
                          <SubIcon />
                        ) : null}
                        <span>{subItem.label}</span>
                        {subItem.tag && <div className="ml-auto">{subItem.tag}</div>}
                      </SidebarMenuSubButton>
                    </NavLink>
                  </SidebarMenuSubItem>
                );
              })}
            </SidebarMenuSub>
          </CollapsibleContent>
        )}
      </SidebarMenuItem>
    </Collapsible>
  );
};
