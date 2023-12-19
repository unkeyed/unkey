import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { vercelIntegrationEnv } from "@/lib/env";
import { Result, result } from "@unkey/result";
import { Vercel } from "@unkey/vercel";
import { z } from "zod";
import { Client } from "./client";
// import { Client } from "./client";

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
    const req = await exchangeCode(props.searchParams.code);
    if (req.error) {
      return <div>error: {JSON.stringify(req.error, null, 2)}</div>;
    }

    integration = {
      id: req.value.installationId,
      workspaceId: workspace.id,
      vercelTeamId: req.value.teamId,
      accessToken: req.value.accessToken,
      createdAt: new Date(),
      deletedAt: null,
    };
    await db.insert(schema.vercelIntegrations).values(integration).execute();
  }
  // return redirect(props.searchParams.next!);
  // return redirect(`/app/settings/vercel?configurationId=${integration.id}`);

  const projects = await new Vercel({
    teamId: integration.vercelTeamId ?? undefined,
    accessToken: integration.accessToken,
  }).listProjects();
  if (projects.error) {
    return (
      <EmptyPlaceholder className="m-8">
        <EmptyPlaceholder.Title>Error</EmptyPlaceholder.Title>
        <EmptyPlaceholder.Description>
          We couldn't load your projects from Vercel. Please try again or contact support.
        </EmptyPlaceholder.Description>
        <Code className="text-left">{JSON.stringify(projects.error, null, 2)}</Code>
      </EmptyPlaceholder>
    );
  }

  if (projects.value.length === 0) {
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
      projects={projects.value}
      apis={workspace.apis}
      returnUrl={props.searchParams.next!}
      integrationId={integration.id}
      accessToken={integration.accessToken}
      vercelTeamId={integration.vercelTeamId}
    />
  );
}

async function exchangeCode(code: string): Promise<
  Result<
    {
      accessToken: string;
      installationId: string;
      userId: string;
      teamId: string | null;
    },
    {
      message: string;
      status?: number;
    }
  >
> {
  try {
    const res = await fetch("https://api.vercel.com/v2/oauth/access_token", {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        client_id: vercelIntegrationEnv()!.VERCEL_INTEGRATION_CLIENT_ID,
        client_secret: vercelIntegrationEnv()!.VERCEL_INTEGRATION_CLIENT_SECRET,
        code,
        redirect_uri: "http://localhost:3000/integrations/vercel/callback",
      }),
    });
    if (!res.ok) {
      return result.fail({
        message: "failed to exchange code for access token",
        status: res.status,
      });
    }

    const data = z
      .object({
        token_type: z.literal("Bearer"),
        access_token: z.string(),
        installation_id: z.string(),
        user_id: z.string(),
        team_id: z.string().nullable(),
      })
      .parse(await res.json());

    return result.success({
      accessToken: data.access_token,
      installationId: data.installation_id,
      userId: data.user_id,
      teamId: data.team_id,
    });
  } catch (e) {
    return result.fail({ message: (e as Error).message });
  }
}
