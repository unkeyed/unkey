"use client";

import { routes } from "@/lib/navigation/routes";
import { ChartUsage, ChevronRight, Gauge } from "@unkey/icons";
import { Item, ItemActions, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";

export function RelatedPages({ workspaceSlug }: { workspaceSlug: string }) {
  const scope = { workspaceSlug };
  const pages: Array<{
    href: Route;
    icon: ReactNode;
    title: string;
    description: string;
  }> = [
    {
      href: routes.settings.usage(scope),
      icon: <ChartUsage />,
      title: "Usage",
      description: "What this workspace used, per project and app.",
    },
    {
      href: routes.settings.limits(scope),
      icon: <Gauge />,
      title: "Limits",
      description: "The ceilings we apply to this workspace.",
    },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {pages.map((page) => (
        <Item key={page.title} variant="outline" render={<Link href={page.href} />}>
          <ItemMedia>{page.icon}</ItemMedia>
          <ItemContent>
            <ItemTitle>{page.title}</ItemTitle>
            <ItemDescription>{page.description}</ItemDescription>
          </ItemContent>
          <ItemActions>
            <ChevronRight />
          </ItemActions>
        </Item>
      ))}
    </div>
  );
}
