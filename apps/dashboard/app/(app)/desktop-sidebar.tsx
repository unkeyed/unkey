"use client";
import { createWorkspaceNavigation, resourcesNavigation } from "@/app/(app)/workspace-navigations";
import { Feedback } from "@/components/dashboard/feedback-component";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useDelayLoader } from "@/hooks/useDelayLoader";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { Loader2, type LucideIcon } from "lucide-react";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import { useRouter } from "next/navigation";
import type React from "react";
import { useTransition } from "react";
// import { WorkspaceSwitcher } from "./team-switcher";
// import { UserButton } from "./user-button";
type Props = {
  workspace: Workspace & {
    apis: {
      id: string;
      name: string;
    }[];
  };
  className?: string;
};

type NavItem = {
  disabled?: boolean;
  tooltip?: string;
  icon: LucideIcon | React.ElementType;
  href: string;
  external?: boolean;
  label: string;
  active?: boolean;
  tag?: React.ReactNode;
  hidden?: boolean;
};

export const DesktopSidebar: React.FC<Props> = ({ workspace, className }) => {
  const segments = useSelectedLayoutSegments() ?? [];
  const workspaceNavigation = createWorkspaceNavigation(workspace, segments);

  const firstOfNextMonth = new Date();
  firstOfNextMonth.setUTCMonth(firstOfNextMonth.getUTCMonth() + 1);
  firstOfNextMonth.setDate(1);
  return (
    <aside
      className={cn(
        "bg-background text-content/65 inset-y-0 w-64 px-5 z-10 h-full shrink-0 flex flex-col overflow-y-auto",
        className,
      )}
    >
      <div className="flex min-w-full mt-2 -mx-2">
        {/* <WorkspaceSwitcher workspace={workspace} /> */}
      </div>
      {workspace.planDowngradeRequest ? (
        <div className="flex justify-center w-full mt-2">
          <Tooltip>
            <TooltipTrigger>
              <Badge size="sm">Subscription ending</Badge>
            </TooltipTrigger>
            <TooltipContent>
              Your plan is schedueld to be downgraded to the {workspace.planDowngradeRequest} tier
              on {firstOfNextMonth.toDateString()}
            </TooltipContent>
          </Tooltip>
        </div>
      ) : null}
      <nav className="flex flex-col flex-1 flex-grow mt-6 pb-10">
        <ul className="flex flex-col flex-1 gap-y-6">
          <li className="flex flex-col gap-2">
            <h2 className="text-xs leading-6 uppercase">Workspace</h2>
            <ul className="-mx-2 space-y-1">
              {workspaceNavigation.map((item) => (
                <li key={item.label}>
                  <NavLink item={item} />
                </li>
              ))}
            </ul>
          </li>
          <li className="flex flex-col gap-2">
            <h2 className="text-xs leading-6 uppercase">Resources</h2>
            <ul className="-mx-2 space-y-1">
              {resourcesNavigation.map((item) => (
                <li key={item.label}>
                  <NavLink item={item} />
                </li>
              ))}
              <li>
                <Feedback />
              </li>
            </ul>
          </li>
        </ul>
      </nav>

      <div className="bg-[inherit] min-w-full [flex:0_0_56px] -mx-2 sticky bottom-0">
        {/* <UserButton /> */}

        {/* Fading indicator that there are more items to scroll */}
        <div className="pointer-events-none absolute bottom-full inset-x-0 h-10 bg-[inherit] [mask-image:linear-gradient(to_top,white,transparent)]" />
      </div>
    </aside>
  );
};

const NavLink: React.FC<{ item: NavItem }> = ({ item }) => {
  const [isPending, startTransition] = useTransition();
  const showLoader = useDelayLoader(isPending);
  const router = useRouter();
  const link = (
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
      className={cn(
        "transition-all duration-150 group flex gap-x-2 rounded-md px-2 py-1 text-sm font-normal leading-6 items-center border border-transparent hover:bg-background-subtle hover:text-content justify-between",
        {
          "bg-background border-border text-content [box-shadow:0px_1px_3px_0px_rgba(0,0,0,0.03)]":
            item.active,
          "text-content-subtle pointer-events-none": item.disabled,
        },
      )}
    >
      <div className="flex items-center group gap-x-2">
        <span className="flex h-5 w-5 shrink-0 items-center justify-center text-[0.625rem]">
          {showLoader ? (
            <Loader2 className="w-5 h-5 shrink-0 animate-spin" />
          ) : (
            <item.icon className="w-5 h-5 shrink-0 [stroke-width:1.25px]" aria-hidden="true" />
          )}
        </span>
        <p className="truncate whitespace-nowrap">{item.label}</p>
      </div>
      {item.tag}
    </Link>
  );

  if (item.tooltip) {
    return (
      <Tooltip>
        <TooltipTrigger className="w-full">
          {link}
          <TooltipContent>{item.tooltip}</TooltipContent>
        </TooltipTrigger>
      </Tooltip>
    );
  }
  return link;
};
