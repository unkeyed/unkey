import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { Activity, BookOpen, Code, Crown, Settings } from "lucide-react";
import React from "react";
import { type NavItem, NavLink } from "./desktop-sidebar-navlink";
import { WorkspaceSwitcher } from "./team-switcher";
// import { UserButton } from "./user-button";
type Props = {
  slug: string;
  workspace: Workspace & {
    apis: {
      id: string;
      name: string;
    }[];
  };
  className?: string;
};

export const DesktopSidebar: React.FC<Props> = ({ slug, workspace, className }) => {
  const navigation: NavItem[] = [
    {
      icon: <Code className="w-4 h-4 shrink-0" aria-hidden="true" />,
      href: "/app/apis",
      label: "APIs",
      active: false,
    },
    {
      icon: <Settings className="w-4 h-4 shrink-0" aria-hidden="true" />,
      href: "/app/settings/general",
      label: "Settings",
      active: false,
    },
    {
      icon: <BookOpen className="w-4 h-4 shrink-0" aria-hidden="true" />,
      href: "https://unkey.dev/docs",
      external: true,
      label: "Docs",
    },
    {
      icon: <Activity className="w-4 h-4 shrink-0" aria-hidden="true" />,
      href: "/app/audit",
      label: "Audit Log",
      disabled: !workspace.betaFeatures.auditLogRetentionDays,
      tooltip:
        "Audit logs are in private beta, please contact support@unkey.dev if you want early access.",
      tag: (
        <div className="bg-background border text-content-subtle rounded text-xs px-1 py-0.5 font-mono">
          beta
        </div>
      ),
    },
  ];
  if (workspace.features.successPage) {
    navigation.push({
      icon: <Crown className="w-4 h-4 shrink-0" aria-hidden="true" />,
      href: "/app/success",
      label: "Success",
      tag: (
        <div className="bg-background border text-content-subtle rounded text-xs px-1 py-0.5 font-mono">
          internal
        </div>
      ),
    });
  }

  const firstOfNextMonth = new Date();
  firstOfNextMonth.setUTCMonth(firstOfNextMonth.getUTCMonth() + 1);
  firstOfNextMonth.setDate(1);

  return (
    <aside className={cn("fixed inset-y-0 w-64 px-6 z-10", className)}>
      <div className="flex min-w-full mt-4 -mx-2">
        <WorkspaceSwitcher slug={slug} />
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
      <nav className="flex flex-col flex-1 flex-grow mt-4">
        <ul className="flex flex-col flex-1 gap-y-7">
          <li>
            <h2 className="text-xs font-semibold leading-6 text-content">General</h2>
            <ul className="mt-2 -mx-2 space-y-1">
              {navigation.map((item) => (
                <li key={item.label}>
                  <NavLink item={item} />
                </li>
              ))}
            </ul>
          </li>
          <li>
            <h2 className="text-xs font-semibold leading-6 text-content">Your APIs</h2>
            {/* max-h-64 in combination with the h-8 on the <TooltipTrigger> will fit 8 apis nicely */}
            <ScrollArea className="mt-2 -mx-2 space-y-1 overflow-auto max-h-64">
              {workspace.apis.map((api) => (
                <Tooltip key={api.id}>
                  <TooltipTrigger className="w-full h-8 overflow-hidden text-ellipsis">
                    <NavLink
                      item={{
                        icon: <Code className="w-4 h-4 shrink-0" />,
                        href: `/app/apis/${api.id}`,
                        label: api.name,
                      }}
                    />
                  </TooltipTrigger>
                  <TooltipContent>{api.name}</TooltipContent>
                </Tooltip>
              ))}
            </ScrollArea>
          </li>
        </ul>
      </nav>

      {/* <UserButton /> */}
    </aside>
  );
};
