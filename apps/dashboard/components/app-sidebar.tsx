"use client";
import { WorkspaceSwitcher } from "@/app/(app)/team-switcher";
import { UserButton } from "@/app/(app)/user-button";
import {
  type NavItem,
  createWorkspaceNavigation,
  resourcesNavigation,
} from "@/app/(app)/workspace-navigations";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { useDelayLoader } from "@/hooks/useDelayLoader";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { SidebarLeftHide, SidebarLeftShow } from "@unkey/icons";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { useRouter, useSelectedLayoutSegments } from "next/navigation";
import { useEffect, useState, useTransition } from "react";

const getButtonStyles = (isActive?: boolean, showLoader?: boolean) => {
  return cn(
    "flex items-center group text-[13px] font-medium text-accent-12 hover:bg-grayA-3 hover:text-accent-12 justify-start active:border focus:ring-2 w-full text-left",
    "rounded-lg transition-colors focus-visible:ring-1 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 disabled:cursor-not-allowed outline-none",
    "focus:border-grayA-12 focus:ring-gray-6 focus-visible:outline-none focus:ring-offset-0 drop-shadow-button",
    isActive ? "bg-grayA-3 text-accent-12" : "[&_svg]:text-gray-9",
    showLoader ? "bg-grayA-3 [&_svg]:text-accent-12" : "",
  );
};

// Function to create navigation items that can have sub-items
const createNestedNavigation = (
  workspace: Pick<Workspace, "features" | "betaFeatures">,
  segments: string[],
): (NavItem & { items?: NavItem[] })[] => {
  // Get the base navigation items
  const baseNav = createWorkspaceNavigation(workspace, segments);
  return baseNav;
};

// Navigation item renderer that supports both regular and nested items
const NavItems = ({ item }: { item: NavItem & { items?: NavItem[] } }) => {
  const [isPending, startTransition] = useTransition();
  const showLoader = useDelayLoader(isPending);
  const router = useRouter();
  // For loading indicators in sub-items
  const [subPending, setSubPending] = useState<Record<string, boolean>>({});
  const Icon = item.icon;

  // Render a flat navigation item (no sub-items)
  if (!item.items || item.items.length === 0) {
    return (
      <SidebarMenuItem>
        <Link
          prefetch
          href={item.href}
          onClick={() => {
            if (!item.external) {
              startTransition(() => {
                router.push(item.href);
              });
            }
          }}
          target={item.external ? "_blank" : undefined}
        >
          <SidebarMenuButton
            tooltip={item.label}
            isActive={item.active}
            className={getButtonStyles(item.active, showLoader)}
          >
            {showLoader ? <AnimatedLoadingSpinner /> : <Icon />}
            <span>{item.label}</span>
          </SidebarMenuButton>
        </Link>
      </SidebarMenuItem>
    );
  }

  // Render a collapsible navigation item with sub-items
  return (
    <Collapsible asChild defaultOpen={item.active} className="group/collapsible">
      <SidebarMenuItem>
        <CollapsibleTrigger asChild>
          <SidebarMenuButton
            tooltip={item.tooltip}
            isActive={item.active}
            className={getButtonStyles(item.active, showLoader)}
          >
            {showLoader ? <AnimatedLoadingSpinner /> : <Icon />}
            <span>{item.label}</span>
            <ChevronRight className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
          </SidebarMenuButton>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <SidebarMenuSub>
            {item.items.map((subItem) => (
              <SidebarMenuSubItem key={subItem.label}>
                <Link
                  prefetch
                  href={subItem.href}
                  onClick={() => {
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
                  }}
                  target={subItem.external ? "_blank" : undefined}
                >
                  <SidebarMenuSubButton
                    isActive={subItem.active}
                    className={getButtonStyles(subItem.active, subPending[subItem.label])}
                  >
                    {subPending[subItem.label] ? (
                      <AnimatedLoadingSpinner />
                    ) : subItem.icon ? (
                      <subItem.icon />
                    ) : null}
                    <span>{subItem.label}</span>
                  </SidebarMenuSubButton>
                </Link>
              </SidebarMenuSubItem>
            ))}
          </SidebarMenuSub>
        </CollapsibleContent>
      </SidebarMenuItem>
    </Collapsible>
  );
};

// AppSidebar component
export function AppSidebar({
  ...props
}: React.ComponentProps<typeof Sidebar> & { workspace: Workspace }) {
  const segments = useSelectedLayoutSegments() ?? [];
  const navItems = createNestedNavigation(props.workspace, segments);

  const { toggleSidebar, state, isMobile } = useSidebar();
  const isCollapsed = state === "collapsed";

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="px-4 mb-1 items-center">
        <div
          className={cn(
            "flex w-full",
            isCollapsed ? "justify-center" : "items-center justify-between gap-4",
          )}
        >
          <WorkspaceSwitcher workspace={props.workspace} />
          {!isMobile && (
            <>
              {!isCollapsed && (
                // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
                <div onClick={toggleSidebar} className="cursor-pointer flex-shrink-0">
                  <SidebarLeftHide className="text-gray-8" size="xl-medium" />
                </div>
              )}
              {isCollapsed && (
                // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
                <div
                  onClick={toggleSidebar}
                  className="absolute -right-3 top-4 cursor-pointer p-1 rounded-full bg-white dark:bg-gray-900 shadow-sm border border-gray-200 dark:border-gray-700"
                >
                  <SidebarLeftShow className="text-gray-8" size="md-medium" />
                </div>
              )}
            </>
          )}
        </div>
      </SidebarHeader>
      <SidebarContent className="px-2">
        <SidebarGroup>
          <SidebarMenu className="gap-2">
            {navItems.map((item) => (
              <NavItems key={item.label} item={item} />
            ))}
            {resourcesNavigation.map((item) => (
              <NavItems key={item.label} item={item} />
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter className={cn("px-4", !isMobile && "items-center")}>
        <UserButton />
      </SidebarFooter>
    </Sidebar>
  );
}

const AnimatedLoadingSpinner = () => {
  const [segmentIndex, setSegmentIndex] = useState(0);
  // Each segment ID in the order they should light up
  const segments = [
    "segment-1", // Right top
    "segment-2", // Right
    "segment-3", // Right bottom
    "segment-4", // Bottom
    "segment-5", // Left bottom
    "segment-6", // Left
    "segment-7", // Left top
    "segment-8", // Top
  ];

  useEffect(() => {
    // Animate the segments in sequence
    const timer = setInterval(() => {
      setSegmentIndex((prevIndex) => (prevIndex + 1) % segments.length);
    }, 125); // 125ms per segment = 1s for full rotation
    return () => clearInterval(timer);
  }, []);

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="18"
      height="18"
      viewBox="0 0 18 18"
      className="animate-spin-slow"
      data-prefers-reduced-motion="respect-motion-preference"
    >
      <g>
        {segments.map((id, index) => {
          // Calculate opacity based on position relative to current index
          const distance = (segments.length + index - segmentIndex) % segments.length;
          const opacity = distance <= 4 ? 1 - distance * 0.2 : 0.1;
          return (
            <path
              key={id}
              id={id}
              style={{
                fill: "currentColor",
                opacity: opacity,
                transition: "opacity 0.12s ease-in-out",
              }}
              d={getPathForSegment(index)}
            />
          );
        })}
        <path
          d="M9,6.5c-1.379,0-2.5,1.121-2.5,2.5s1.121,2.5,2.5,2.5,2.5-1.121,2.5-2.5-1.121-2.5-2.5-2.5Z"
          style={{ fill: "currentColor", opacity: 0.6 }}
        />
      </g>
    </svg>
  );
};

// Helper function to get path data for each segment
function getPathForSegment(index: number) {
  const paths = [
    "M13.162,3.82c-.148,0-.299-.044-.431-.136-.784-.552-1.662-.915-2.61-1.08-.407-.071-.681-.459-.61-.867,.071-.408,.459-.684,.868-.61,1.167,.203,2.248,.65,3.216,1.33,.339,.238,.42,.706,.182,1.045-.146,.208-.378,.319-.614,.319Z",
    "M16.136,8.5c-.357,0-.675-.257-.738-.622-.163-.942-.527-1.82-1.082-2.608-.238-.339-.157-.807,.182-1.045,.34-.239,.809-.156,1.045,.182,.683,.97,1.132,2.052,1.334,3.214,.07,.408-.203,.796-.611,.867-.043,.008-.086,.011-.129,.011Z",
    "M14.93,13.913c-.148,0-.299-.044-.431-.137-.339-.238-.42-.706-.182-1.045,.551-.784,.914-1.662,1.078-2.609,.071-.408,.466-.684,.867-.611,.408,.071,.682,.459,.611,.867-.203,1.167-.65,2.25-1.33,3.216-.146,.208-.378,.318-.614,.318Z",
    "M10.249,16.887c-.357,0-.675-.257-.738-.621-.07-.408,.202-.797,.61-.868,.945-.165,1.822-.529,2.608-1.082,.34-.238,.807-.156,1.045,.182,.238,.338,.157,.807-.182,1.045-.968,.682-2.05,1.13-3.214,1.333-.044,.008-.087,.011-.13,.011Z",
    "M7.751,16.885c-.043,0-.086-.003-.13-.011-1.167-.203-2.249-.651-3.216-1.33-.339-.238-.42-.706-.182-1.045,.236-.339,.702-.421,1.045-.183,.784,.551,1.662,.915,2.61,1.08,.408,.071,.681,.459,.61,.868-.063,.364-.381,.621-.738,.621Z",
    "M3.072,13.911c-.236,0-.469-.111-.614-.318-.683-.97-1.132-2.052-1.334-3.214-.07-.408,.203-.796,.611-.867,.403-.073,.796,.202,.867,.61,.163,.942,.527,1.82,1.082,2.608,.238,.339,.157,.807-.182,1.045-.131,.092-.282,.137-.431,.137Z",
    "M1.866,8.5c-.043,0-.086-.003-.129-.011-.408-.071-.682-.459-.611-.867,.203-1.167,.65-2.25,1.33-3.216,.236-.339,.703-.422,1.045-.182,.339,.238,.42,.706,.182,1.045-.551,.784-.914,1.662-1.078,2.609-.063,.365-.381,.622-.738,.622Z",
    "M4.84,3.821c-.236,0-.468-.111-.614-.318-.238-.338-.157-.807,.182-1.045,.968-.682,2.05-1.13,3.214-1.333,.41-.072,.797,.202,.868,.61,.07,.408-.202,.797-.61,.868-.945,.165-1.822,.529-2.608,1.082-.131,.092-.282,.137-.431,.137Z",
  ];
  return paths[index];
}

// Add CSS for the spin-slow animation
if (typeof document !== "undefined") {
  const style = document.createElement("style");
  style.textContent = `
    @media (prefers-reduced-motion: reduce) {
      [data-prefers-reduced-motion="respect-motion-preference"] {
        animation: none !important;
        transition: none !important;
      }
    }
    
    @keyframes spin-slow {
      from {
        transform: rotate(0deg);
      }
      to {
        transform: rotate(360deg);
      }
    }
    
    .animate-spin-slow {
      animation: spin-slow 1.5s linear infinite;
    }
  `;
  document.head.appendChild(style);
}
