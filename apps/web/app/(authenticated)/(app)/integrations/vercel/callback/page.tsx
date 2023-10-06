import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Result, result } from "@/lib/errors";
import { redirect } from "next/navigation";
import { z } from "zod";
// import { Client } from "./client";

type Props = {
  searchParams: {
    code?: string;
    next?: string;
  };
};
export default async function Page(props: Props) {
  if (!props.searchParams.code) {
    return <div>no code</div>;
  }

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, getTenantId()),
    with: {
      apis: true,
    },
  });
  if (!workspace) {
    return <div>no workspace</div>;
  }

  const req = await exchangeCode(props.searchParams.code);
  if (req.error) {
    return <div>error: {JSON.stringify(req.error, null, 2)}</div>;
  }

  let integration = await db.query.vercelIntegrations.findFirst({
    where: eq(schema.vercelIntegrations.id, req.value.installationId),
  });
  if (!integration) {
    integration = {
      id: req.value.installationId,
      workspaceId: workspace.id,
      vercelTeamId: req.value.teamId,
      accessToken: req.value.accessToken,
    };
    await db.insert(schema.vercelIntegrations).values(integration).execute();
  }
  return redirect(props.searchParams.next!);
  // return redirect(`/app/settings/vercel?configurationId=${integration.id}`);

  // const projects = await getProjects(req.value.accessToken, req.value.teamId);

  // if (projects.error) {
  //   return <div>error: {JSON.stringify(projects.error, null, 2)}</div>;
  // }
  // if (projects.value.length === 0) {
  //   return <div>no projects</div>;
  // }
  // return (
  //   <Client
  //     projects={projects.value}
  //     apis={workspace.apis}
  //     returnUrl={props.searchParams.next!}
  //     integrationId={integration.id}
  //   />
  // );
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
        client_id: env().VERCEL_INTEGRATION_CLIENT_ID,
        client_secret: env().VERCEL_INTEGRATION_CLIENT_SECRET,
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
