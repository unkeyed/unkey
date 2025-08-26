"use client";

import { cn } from "@/lib/utils";
import { Cloud, Gear, GridCircle, Layers3 } from "@unkey/icons";
import type { IconProps } from "@unkey/icons/src/props";
import { Button } from "@unkey/ui";
import { useParams, usePathname, useRouter } from "next/navigation";

type TabItem = {
  id: string;
  label: string;
  icon?: React.ComponentType<IconProps>;
  path: string;
};

export const ProjectSubNavigation = () => {
  const router = useRouter();
  const params = useParams();
  const pathname = usePathname();
  const projectId = params?.projectId as string;

  // Detect current route and set active tab
  const getCurrentTab = (): string => {
    const segments = pathname?.split("/");

    if (!segments) {
      throw new Error("URL Segments are empty.");
    }

    const tabIndex = segments.findIndex((segment) => segment === projectId) + 1;
    const currentTab = segments[tabIndex];

    const validTabs = ["overview", "deployments", "logs", "settings"];
    return validTabs.includes(currentTab) ? currentTab : "overview";
  };

  const activeTab = getCurrentTab();

  const tabs: TabItem[] = [
    {
      id: "overview",
      label: "Overview",
      icon: GridCircle,
      path: `/projects/${projectId}`,
    },
    {
      id: "deployments",
      label: "Deployments",
      icon: Cloud,
      path: `/projects/${projectId}/deployments`,
    },
    {
      id: "logs",
      label: "Logs",
      icon: Layers3,
      path: `/projects/${projectId}/logs`,
    },
    {
      id: "settings",
      label: "Settings",
      icon: Gear,
      path: `/projects/${projectId}/settings`,
    },
  ];

  const handleTabChange = (path: string) => {
    router.push(path);
  };

  if (!projectId) {
    throw new Error("ProjectSubNavigation requires a valid project ID");
  }

  return (
    <div className="w-full border-b border-gray-4 bg-transparent relative">
      <div className="flex">
        {tabs.map((tab) => {
          const IconComponent = tab.icon || GridCircle;
          const isActive = tab.id === activeTab;
          return (
            <Button
              key={tab.id}
              variant="ghost"
              onClick={() => handleTabChange(tab.path)}
              className={cn(
                "flex gap-2.5 items-center px-5 py-2 min-h-[40px] relative text-[13px] leading-4 font-medium focus:ring-0 transition-colors duration-150 rounded-none",
                "after:absolute after:-bottom-[1px] after:left-0 after:right-0 after:h-px after:bg-accent-12 after:scale-x-0 after:transition-transform after:duration-300 after:ease-out",
                isActive && "text-accent-12 after:scale-x-100",
              )}
            >
              <IconComponent size="sm-medium" />
              <span>{tab.label}</span>
            </Button>
          );
        })}
      </div>
    </div>
  );
};
