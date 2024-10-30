import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { vercelIntegrationEnv } from "@/lib/env";
import { Vercel } from "@unkey/vercel";
import { Client } from "./client";
import { exchangeCode } from "./exchange-code";

type Props = {
  searchParams: {
    code?: string;
    next?: string;
    configurationId: string;
  };
};

export default async function Page(props: Props) {
  const vercelEnv = vercelIntegrationEnv();
  if (!vercelEnv) {
    return <div>Set up env</div>;
  }

  if (!props.searchParams.code) {
    return <div>no code</div>;
  }

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, getTenantId()), isNull(table.deletedAt)),
    with: {
      apis: { where: (table, { isNull }) => isNull(table.deletedAt) },
    },
  });

  if (!workspace) {
    return <div>no workspace</div>;
  }

  let integration = await db.query.vercelIntegrations.findFirst({
    where: eq(schema.vercelIntegrations.id, props.searchParams.configurationId),
  });

  if (!integration) {
    const { val, err } = await exchangeCode(props.searchParams.code);
    if (err) {
      return <div>error: {JSON.stringify(err, null, 2)}</div>;
    }

    integration = {
      id: val.installationId,
      workspaceId: workspace.id,
      vercelTeamId: val.teamId,
      accessToken: val.accessToken,
      createdAt: new Date(),
      deletedAt: null,
    };
    await db.insert(schema.vercelIntegrations).values(integration).execute();
  }
  // return redirect(props.searchParams.next!);
  // return redirect(`/settings/vercel?configurationId=${integration.id}`);

  const projects = await new Vercel({
    teamId: integration.vercelTeamId ?? undefined,
    accessToken: integration.accessToken,
  }).listProjects();
  if (projects.err) {
    return (
      <EmptyPlaceholder className="m-8">
        <EmptyPlaceholder.Title>Error</EmptyPlaceholder.Title>
        <EmptyPlaceholder.Description>
          We couldn't load your projects from Vercel. Please try again or contact support.
        </EmptyPlaceholder.Description>
        <Code className="text-left">
          {JSON.stringify(
            {
              message: projects.err.message,
              context: projects.err.context,
            },
            null,
            2,
          )}
        </Code>
      </EmptyPlaceholder>
    );
  }

  if (projects.val.length === 0) {
    return (
      <EmptyPlaceholder className="m-8">
        <EmptyPlaceholder.Title>No Projects Found</EmptyPlaceholder.Title>
        <EmptyPlaceholder.Description>
          You did not authorize any projects to be connected. Please go to your Vercel dashboard and
          add a project to this integration.
        </EmptyPlaceholder.Description>
      </EmptyPlaceholder>
    );
  }
  return (
    <Client
      projects={projects.val}
      apis={workspace.apis}
      returnUrl={props.searchParams.next!}
      integrationId={integration.id}
      accessToken={integration.accessToken}
      vercelTeamId={integration.vercelTeamId}
    />
  );
}
