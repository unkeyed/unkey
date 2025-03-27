import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import type { NavItem } from "../../../workspace-navigations";
import { getButtonStyles } from "./utils";

export const ToggleSidebarButton = ({
  toggleNavItem,
  toggleSidebar,
}: {
  toggleNavItem: NavItem;
  toggleSidebar: () => void;
}) => {
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        tooltip={toggleNavItem.tooltip}
        isActive={toggleNavItem.active}
        className={getButtonStyles(toggleNavItem.active)}
        onClick={toggleSidebar}
      >
        {toggleNavItem.icon && <toggleNavItem.icon size="xl-medium" />}
        <span>{toggleNavItem.label}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
};
