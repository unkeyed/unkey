"use client";

import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { ChevronExpandY, Cloud } from "@unkey/icons";
import { useDeploymentBreadcrumbConfig } from "./use-deployment-breadcrumb-config";

export function DeploymentNavbar() {
  const breadcrumbs = useDeploymentBreadcrumbConfig();

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cloud />}>
        {breadcrumbs
          .filter((breadcrumb) => breadcrumb.shouldRender)
          .map((breadcrumb) => (
            <Navbar.Breadcrumbs.Link
              key={breadcrumb.id}
              href={breadcrumb.href}
              noop={breadcrumb.noop}
              isIdentifier={breadcrumb.isIdentifier}
              active={breadcrumb.active}
            >
              {breadcrumb.quickNavConfig ? (
                <QuickNavPopover
                  items={breadcrumb.quickNavConfig.items}
                  activeItemId={breadcrumb.quickNavConfig.activeItemId}
                  shortcutKey={breadcrumb.quickNavConfig.shortcutKey}
                >
                  <div className="flex items-center gap-1">
                    {breadcrumb.children}
                    <ChevronExpandY className="size-4" />
                  </div>
                </QuickNavPopover>
              ) : (
                breadcrumb.children
              )}
            </Navbar.Breadcrumbs.Link>
          ))}
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
