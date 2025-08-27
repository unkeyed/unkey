/**
 * Deprecated with new auth
 * Hiding for now until we decide if we want to fix it up or toss it
 */

import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navigation } from "@/components/navigation/navigation";
import { PageContent } from "@/components/page-content";
import { getAuth } from "@/lib/auth";
import { auth } from "@/lib/auth/server";
import { type Api, type Key, type VercelBinding, db, eq, schema } from "@/lib/db";
import { Gear } from "@unkey/icons";
import { Button, Code, Empty } from "@unkey/ui";
import { Vercel } from "@unkey/vercel";
import Link from "next/link";
import { notFound } from "next/navigation";
import { navigation } from "../constants";
import { Client } from "./client";

export const dynamic = "force-dynamic";

type Props = {
  searchParams: {
    configurationId?: string;
  };
};

export default async function Page(props: Props) {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
      },
      vercelIntegrations: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
        with: {
          vercelBindings: {
            where: (table, { isNull }) => isNull(table.deletedAtM),
          },
        },
      },
    },
  });

  if (!workspace) {
    console.warn("no workspace");
    return notFound();
  }

  const integration = props.searchParams.configurationId
    ? workspace.vercelIntegrations.find((i) => i.id === props.searchParams.configurationId)
    : workspace.vercelIntegrations.at(0);

  if (!integration) {
    return (
      <div>
        <Navigation href={`/${workspace.id}/settings/vercel`} name="Settings" icon={<Gear />} />
        <PageContent>
          <SubMenu navigation={navigation} segment="vercel" />
          <div className="mt-8" />
          <Empty>
            <Empty.Title>Vercel is not connected to this workspace</Empty.Title>
            <Empty.Actions>
              <Link target="_blank" href="https://vercel.com/integrations/unkey">
                <Button>Connect</Button>
              </Link>
            </Empty.Actions>
          </Empty>
        </PageContent>
      </div>
    );
  }

  const vercel = new Vercel({
    accessToken: integration.accessToken,
    teamId: integration.vercelTeamId ?? undefined,
  });

  const { val: rawProjects, err } = await vercel.listProjects();

  if (err) {
    return (
      <Empty>
        <Empty.Title>Error</Empty.Title>
        <Empty.Description>
          We couldn't load your projects from Vercel. Please try again or contact support.
        </Empty.Description>
        <Empty.Description>
          <Code className="text-left">{JSON.stringify(err, null, 2)}</Code>
        </Empty.Description>
      </Empty>
    );
  }

  const apis = workspace.apis.reduce(
    (acc, api) => {
      acc[api.id] = api;
      return acc;
    },
    {} as Record<string, Api>,
  );

  const rootKeys = (
    await db.query.keys.findMany({
      where: eq(schema.keys.forWorkspaceId, workspace.id),
    })
  ).reduce(
    (acc, key) => {
      acc[key.id] = key;
      return acc;
    },
    {} as Record<string, Key>,
  );

  const users = (
    await Promise.all(
      [...new Set(integration.vercelBindings.map((b) => b.lastEditedBy))]
        .filter(Boolean)
        .map(async (id) => {
          try {
            const u = await auth.getUser(id);
            if (!u) {
              console.error(`User not found for ID: ${id}`);
              return null;
            }
            return {
              id: u.id,
              name: u.fullName || u.email || "Unknown User",
              image: u.avatarUrl,
            };
          } catch (error) {
            console.error(`Failed to fetch user ${id}:`, error);
            return null;
          }
        }),
    )
  )
    .filter((user): user is NonNullable<typeof user> => user !== null)
    .reduce(
      (acc, user) => {
        acc[user.id] = user;
        return acc;
      },
      {} as Record<string, { id: string; name: string; image: string | null }>,
    );

  const projects = await Promise.all(
    rawProjects.map(async (p) => ({
      id: p.id,
      name: p.name,
      bindings: integration.vercelBindings
        .filter((binding) => binding.projectId === p.id)
        .reduce(
          (acc, binding) => {
            if (!acc[binding.environment]) {
              acc[binding.environment] = {
                apiId: null,
                rootKey: null,
              };
            }
            acc[binding.environment][binding.resourceType] = {
              ...binding,
              updatedBy: users[binding.lastEditedBy],
            };
            return acc;
          },
          {} as Record<
            VercelBinding["environment"],
            Record<
              VercelBinding["resourceType"],
              | (VercelBinding & {
                  updatedBy: { id: string; name: string; image: string | null };
                })
              | null
            >
          >,
        ),
    })),
  );

  return (
    <div>
      <Navigation href={`/${workspace.id}/settings/vercel`} icon={<Gear />} name="Settings" />
      <PageContent>
        <SubMenu navigation={navigation} segment="vercel" />
        <div className="mt-8" />
        <Client projects={projects} apis={apis} rootKeys={rootKeys} integration={integration} />
      </PageContent>
    </div>
  );
}
