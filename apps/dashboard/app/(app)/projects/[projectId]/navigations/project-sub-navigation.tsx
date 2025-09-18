"use client";

import { cn } from "@/lib/utils";
import { Cloud, GridCircle, Layers3 } from "@unkey/icons";
import type { IconProps } from "@unkey/icons/src/props";
import { useParams, usePathname, useRouter } from "next/navigation";
import { useEffect, useRef } from "react";

type TabItem = {
  id: string;
  label: string;
  icon?: React.ComponentType<IconProps>;
  path: string;
};

export const ProjectSubNavigation = ({
  onMount,
  detailsExpandableTrigger,
}: {
  onMount: (distanceToTop: number) => void;
  detailsExpandableTrigger: React.ReactNode;
}) => {
  const router = useRouter();
  const params = useParams();
  const pathname = usePathname();
  const projectId = params?.projectId as string;

  const anchorRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (onMount) {
      const distanceToTop = anchorRef.current?.getBoundingClientRect().top ?? 0;
      onMount(distanceToTop);
    }
  }, [onMount]);

  // Detect current route and set active tab
  const getCurrentTab = (): string => {
    const segments = pathname?.split("/");

    if (!segments) {
      throw new Error("URL Segments are empty.");
    }

    const tabIndex = segments.findIndex((segment) => segment === projectId) + 1;
    const currentTab = segments[tabIndex];

    const validTabs = ["overview", "deployments", "gateway-logs", "settings"];
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
      id: "gateway-logs",
      label: "Gateway Logs",
      icon: Layers3,
      path: `/projects/${projectId}/logs`,
    },
  ];

  const handleTabChange = (path: string) => {
    router.push(path);
  };

  if (!projectId) {
    throw new Error("ProjectSubNavigation requires a valid project ID");
  }

  return (
    <div className="w-full border-b border-gray-4 bg-transparent relative h-10" ref={anchorRef}>
      <div className="flex items-center h-full">
        {tabs.map((tab) => {
          const IconComponent = tab.icon || GridCircle;
          const isActive = tab.id === activeTab;
          return (
            // Our <Button /> component has too many default options using a native button easier to use in this case.
            <button
              type="button"
              key={tab.id}
              onClick={() => handleTabChange(tab.path)}
              className={cn(
                "flex gap-2.5 items-center px-5 py-2 h-full relative text-[13px] leading-4 font-medium transition-colors duration-150 rounded-none",
                "text-gray-12 bg-transparent hover:bg-grayA-4 focus:outline-none",
                "after:absolute after:-bottom-[1px] after:left-0 after:right-0 after:h-px after:bg-accent-12 after:scale-x-0 after:transition-transform after:duration-300 after:ease-out",
                isActive && "text-accent-12 after:scale-x-100",
              )}
            >
              <IconComponent size="sm-medium" />
              <span>{tab.label}</span>
            </button>
          );
        })}
        <div className="ml-auto mr-2 py-2">{detailsExpandableTrigger}</div>
      </div>
    </div>
  );
};
