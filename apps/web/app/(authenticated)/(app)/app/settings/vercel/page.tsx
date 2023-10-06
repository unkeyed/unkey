import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Button } from "@/components/ui/button";
import { getTenantId } from "@/lib/auth";
import { Api, Key, VercelBinding, db, eq, schema } from "@/lib/db";
import { clerkClient } from "@clerk/nextjs";
import { Vercel } from "@unkey/vercel";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Client } from "./client";
type Props = {
  searchParams: {
    configurationId?: string;
  };
};
export default async function Page(props: Props) {
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, getTenantId()),
    with: {
      apis: true,
      vercelIntegrations: {
        with: {
          vercelBindings: true,
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
      <EmptyPlaceholder>
        <EmptyPlaceholder.Title>Vercel is not connected to this workspace</EmptyPlaceholder.Title>
        <EmptyPlaceholder.Description>
          <Link target="_blank" href="https://vercel.com/integrations/unkey-dev">
            <Button>Connect</Button>
          </Link>
        </EmptyPlaceholder.Description>
      </EmptyPlaceholder>
    );
  }

  const vercel = new Vercel({
    accessToken: integration.accessToken,
    teamId: integration.vercelTeamId ?? undefined,
  });

  const rawProjects = await vercel.listProjects();

  const apis = workspace.apis.reduce((acc, api) => {
    acc[api.id] = api;
    return acc;
  }, {} as Record<string, Api>);

  const rootKeys = (
    await db.query.keys.findMany({
      where: eq(schema.keys.forWorkspaceId, workspace.id),
    })
  ).reduce((acc, key) => {
    acc[key.id] = key;
    return acc;
  }, {} as Record<string, Key>);

  const users = (
    await Promise.all(
      [...new Set(integration.vercelBindings.map((b) => b.lastEditedBy))].map(async (id) => {
        const u = await clerkClient.users.getUser(id);
        return {
          id: u.id,
          name: u.username ?? u.emailAddresses.at(0)?.emailAddress ?? "",
          image: u.imageUrl,
        };
      }),
    )
  ).reduce((acc, user) => {
    acc[user.id] = user;
    return acc;
  }, {} as Record<string, { id: string; name: string; image: string }>);

  const projects = await Promise.all(
    rawProjects.map(async (p) => ({
      id: p.id,
      name: p.name,
      bindings: integration.vercelBindings
        .filter((binding) => binding.projectId === p.id)
        .reduce((acc, binding) => {
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
        }, {} as Record<
          VercelBinding["environment"],
          Record<
            VercelBinding["resourceType"],
            (VercelBinding & { updatedBy: { id: string; name: string; image: string } }) | null
          >
        >),
    })),
  );

  return <Client projects={projects} apis={apis} rootKeys={rootKeys} integration={integration} />;
}
