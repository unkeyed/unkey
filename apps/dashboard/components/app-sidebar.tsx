"use client";
import { WorkspaceSwitcher } from "@/components/navigation/sidebar/team-switcher";
import { UserButton } from "@/components/navigation/sidebar/user-button";
import {
  type NavItem,
  createWorkspaceNavigation,
  resourcesNavigation,
} from "@/components/navigation/sidebar/workspace-navigations";
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
import { useDelayLoader } from "@/hooks/use-delay-loader";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { SidebarLeftHide, SidebarLeftShow } from "@unkey/icons";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { useRouter, useSelectedLayoutSegments } from "next/navigation";
import { memo, useEffect, useMemo, useState, useTransition } from "react";

export function AppSidebar({
  ...props
}: React.ComponentProps<typeof Sidebar> & { workspace: Workspace }) {
  const segments = useSelectedLayoutSegments() ?? [];
  const navItems = useMemo(
    () => createNestedNavigation(props.workspace, segments),
    [props.workspace, segments],
  );

  // Create a toggle sidebar nav item
  const toggleNavItem: NavItem = useMemo(
    () => ({
      label: "Toggle Sidebar",
      href: "#",
      icon: SidebarLeftShow,
      active: false,
      tooltip: "Toggle Sidebar",
    }),
    [],
  );

  const { state, isMobile, toggleSidebar } = useSidebar();
  const isCollapsed = state === "collapsed";

  const headerContent = useMemo(
    () => (
      <div
        className={cn(
          "flex w-full",
          isCollapsed ? "justify-center" : "items-center justify-between gap-4",
        )}
      >
        <WorkspaceSwitcher workspace={props.workspace} />
        {state !== "collapsed" && !isMobile && (
          <button type="button" onClick={toggleSidebar}>
            <SidebarLeftHide className="text-gray-8" size="xl-medium" />
          </button>
        )}
      </div>
    ),
    [isCollapsed, props.workspace, state, isMobile, toggleSidebar],
  );

  const resourceNavItems = useMemo(() => resourcesNavigation, []);

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="px-4 mb-1 items-center pt-4">{headerContent}</SidebarHeader>
      <SidebarContent className="px-2">
        <SidebarGroup>
          <SidebarMenu className="gap-2">
            {/* Toggle button as NavItem */}
            {state === "collapsed" && (
              <ToggleSidebarButton toggleNavItem={toggleNavItem} toggleSidebar={toggleSidebar} />
            )}

            {navItems.map((item) => (
              <NavItems key={item.label} item={item} />
            ))}
            {resourceNavItems.map((item) => (
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

const getButtonStyles = (isActive?: boolean, showLoader?: boolean) => {
  return cn(
    "flex items-center group text-[13px] font-medium text-accent-12 hover:bg-grayA-3 hover:text-accent-12 justify-start active:border focus:ring-2 w-full text-left",
    "rounded-lg transition-colors focus-visible:ring-1 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 disabled:cursor-not-allowed outline-none",
    "focus:border-grayA-12 focus:ring-gray-6 focus-visible:outline-none focus:ring-offset-0 drop-shadow-button",
    isActive ? "bg-grayA-3 text-accent-12" : "[&_svg]:text-gray-9",
    showLoader ? "bg-grayA-3 [&_svg]:text-accent-12" : "",
  );
};

const createNestedNavigation = (
  workspace: Pick<Workspace, "features" | "betaFeatures">,
  segments: string[],
): (NavItem & { items?: NavItem[] })[] => {
  const baseNav = createWorkspaceNavigation(workspace, segments);
  return baseNav;
};

const NavLink = memo(
  ({
    href,
    external,
    onClick,
    children,
  }: {
    href: string;
    external?: boolean;
    onClick?: () => void;
    children: React.ReactNode;
  }) => {
    return (
      <Link
        prefetch={!external}
        href={href}
        onClick={onClick}
        target={external ? "_blank" : undefined}
      >
        {children}
      </Link>
    );
  },
);

const SEGMENTS = [
  "segment-1", // Right top
  "segment-2", // Right
  "segment-3", // Right bottom
  "segment-4", // Bottom
  "segment-5", // Left bottom
  "segment-6", // Left
  "segment-7", // Left top
  "segment-8", // Top
];

const AnimatedLoadingSpinner = memo(() => {
  const [segmentIndex, setSegmentIndex] = useState(0);

  useEffect(() => {
    // Animate the segments in sequence
    const timer = setInterval(() => {
      setSegmentIndex((prevIndex) => (prevIndex + 1) % SEGMENTS.length);
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
        {SEGMENTS.map((id, index) => {
          const distance = (SEGMENTS.length + index - segmentIndex) % SEGMENTS.length;
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
});

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

const FlatNavItem = memo(({ item }: { item: NavItem }) => {
  const [isPending, startTransition] = useTransition();
  const showLoader = useDelayLoader(isPending);
  const router = useRouter();
  const Icon = item.icon;

  const handleClick = () => {
    if (!item.external) {
      startTransition(() => {
        router.push(item.href);
      });
    }
  };

  return (
    <SidebarMenuItem>
      <NavLink href={item.href} external={item.external} onClick={handleClick}>
        <SidebarMenuButton
          tooltip={item.label}
          isActive={item.active}
          className={getButtonStyles(item.active, showLoader)}
        >
          {showLoader ? <AnimatedLoadingSpinner /> : <Icon size="xl-medium" />}
          <span>{item.label}</span>
        </SidebarMenuButton>
      </NavLink>
    </SidebarMenuItem>
  );
});

const NestedNavItem = memo(({ item }: { item: NavItem & { items?: NavItem[] } }) => {
  const [isPending, startTransition] = useTransition();
  const showLoader = useDelayLoader(isPending);
  const router = useRouter();
  const Icon = item.icon;

  // For loading indicators in sub-items
  const [subPending, setSubPending] = useState<Record<string, boolean>>({});

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
            {item.items?.map((subItem) => {
              const SubIcon = subItem.icon;

              const handleSubItemClick = () => {
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
                    </SidebarMenuSubButton>
                  </NavLink>
                </SidebarMenuSubItem>
              );
            })}
          </SidebarMenuSub>
        </CollapsibleContent>
      </SidebarMenuItem>
    </Collapsible>
  );
});

const NavItems = memo(({ item }: { item: NavItem & { items?: NavItem[] } }) => {
  if (!item.items || item.items.length === 0) {
    return <FlatNavItem item={item} />;
  }
  return <NestedNavItem item={item} />;
});

const ToggleSidebarButton = memo(
  ({
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
          <toggleNavItem.icon size="xl-medium" />
          <span>{toggleNavItem.label}</span>
        </SidebarMenuButton>
      </SidebarMenuItem>
    );
  },
);

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
