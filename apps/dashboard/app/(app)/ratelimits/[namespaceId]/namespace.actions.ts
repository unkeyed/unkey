"use server";

import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound, redirect } from "next/navigation";

export const getWorkspaceDetails = async (namespaceId: string, fallbackUrl = "/ratelimits") => {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    columns: {
      name: true,
      orgId: true,
    },
    with: {
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
        columns: {
          id: true,
          workspaceId: true,
          name: true,
        },
      },
    },
  });

  if (!workspace || workspace.orgId !== orgId) {
    // Will take users to onboarding
    return redirect("/new");
  }

  const namespace = workspace?.ratelimitNamespaces.find((r) => r.id === namespaceId);

  if (!namespace) {
    return fallbackUrl ? redirect(fallbackUrl) : notFound();
  }

  return { namespace, ratelimitNamespaces: workspace?.ratelimitNamespaces };
};

export const getWorkspaceDetailsWithOverrides = async (
  namespaceId: string,
  fallbackUrl = "/ratelimits",
) => {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    columns: {
      name: true,
      orgId: true,
      features: true,
    },
    with: {
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
        columns: {
          id: true,
          workspaceId: true,
          name: true,
        },
        with: {
          overrides: {
            columns: {
              id: true,
              identifier: true,
              limit: true,
              duration: true,
              async: true,
            },
            where: (table, { isNull }) => isNull(table.deletedAtM),
          },
        },
      },
    },
  });

  if (!workspace || workspace.orgId !== orgId) {
    // Will take users to onboarding
    return redirect("/new");
  }

  const namespace = workspace?.ratelimitNamespaces.find((r) => r.id === namespaceId);

  if (!namespace) {
    return fallbackUrl ? redirect(fallbackUrl) : notFound();
  }

  return { namespace, workspace };
};
