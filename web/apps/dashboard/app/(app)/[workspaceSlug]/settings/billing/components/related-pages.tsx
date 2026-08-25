"use client";

import { routes } from "@/lib/navigation/routes";
import { BILLING_DOCS } from "@/lib/support";
import { BookOpen, ChartUsage } from "@unkey/icons";
import { Item, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";

export function RelatedPages({ workspaceSlug }: { workspaceSlug: string }) {
  const pages: Array<{
    href: Route;
    icon: ReactNode;
    title: string;
    description: string;
    external?: boolean;
  }> = [
    {
      href: routes.settings.usage({ workspaceSlug }),
      icon: <ChartUsage />,
      title: "Usage",
      description: "Track your spend and usage across Unkey",
    },
    {
      href: BILLING_DOCS,
      icon: <BookOpen />,
      title: "Documentation",
      description: "How plans, usage and invoices work",
      external: true,
    },
  ];

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {pages.map((page) => (
        <Item
          key={page.title}
          variant="outline"
          render={
            <Link
              href={page.href}
              target={page.external ? "_blank" : undefined}
              rel={page.external ? "noopener noreferrer" : undefined}
            />
          }
        >
          <ItemMedia>{page.icon}</ItemMedia>
          <ItemContent>
            <ItemTitle>{page.title}</ItemTitle>
            <ItemDescription>{page.description}</ItemDescription>
          </ItemContent>
        </Item>
      ))}
    </div>
  );
}
