import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { notFound } from "next/navigation";
import { Client } from "./client";
import { clerkClient } from "@clerk/nextjs";
import { User } from "@clerk/nextjs/dist/types/server";

type Props = {
  searchParams: {
    configurationId?: string;
  };
};
export default async function Page(props: Props) {
  if (!props.searchParams.configurationId) {
    return notFound();
  }
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, getTenantId()),
    with: {
      apis: true,
      vercelIntegrations: {
        where: eq(schema.vercelIntegrations.id, props.searchParams.configurationId),
        with: {
          vercelBindings: true,
        },
      },
    },
  });
  if (!workspace) {
    return notFound();
  }
  const integration = workspace.vercelIntegrations.at(0);
  if (!integration) {
    return notFound();
  }

  const users = (await Promise.all(integration.vercelBindings.map((binding) => clerkClient.users.getUser(binding.lastEditedBy)))).reduce((acc, user) => {
    acc[user.id] = {
      id: user.id,
      name: user.username?? user.emailAddresses.at(0)?.emailAddress??"",
      image: user.imageUrl,
    }
    return acc
  }, {} as Record<string, { id: string, name: string, image: string }>)

  return (
    <div>
      <Client bindings={integration.vercelBindings} users={users} apis={workspace.apis}/>
    </div>
  );
}
