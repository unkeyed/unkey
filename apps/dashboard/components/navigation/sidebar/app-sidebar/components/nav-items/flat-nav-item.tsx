import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import { useDelayLoader } from "@/hooks/use-delay-loader";
import { cn } from "@/lib/utils";
import { useRouter } from "next/navigation";
import { useTransition } from "react";
import type { NavProps } from ".";
import { NavLink } from "../nav-link";
import { AnimatedLoadingSpinner } from "./animated-loading-spinner";
import { getButtonStyles } from "./utils";

export const FlatNavItem = ({ item, onLoadMore, className }: NavProps) => {
  const [isPending, startTransition] = useTransition();
  const showLoader = useDelayLoader(isPending);
  const router = useRouter();
  const Icon = item.icon;

  const isLoadMoreButton = item.loadMoreAction === true;

  const handleClick = () => {
    if (isLoadMoreButton && onLoadMore) {
      onLoadMore(item);
      return;
    }

    if (!item.external) {
      startTransition(() => {
        router.push(item.href);
      });
    }
  };

  return (
    <SidebarMenuItem className={cn("list-none", className)}>
      <NavLink
        href={item.href}
        external={item.external}
        onClick={handleClick}
        isLoadMoreButton={isLoadMoreButton}
      >
        <SidebarMenuButton
          tooltip={item.tooltip}
          isActive={item.active}
          className={getButtonStyles(item.active, showLoader)}
        >
          {showLoader ? (
            <AnimatedLoadingSpinner />
          ) : Icon ? (
            <Icon iconsize="xl-medium" />
          ) : null}
          <span>{item.label}</span>
          {item.tag && <div className="ml-auto">{item.tag}</div>}
        </SidebarMenuButton>
      </NavLink>
    </SidebarMenuItem>
  );
};
