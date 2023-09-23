import { CopyButton } from "@/components/dashboard/copy-button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { notFound } from "next/navigation";
import { DeleteKey } from "./delete-key";
import { UpdateKeyExpiration } from "./update-key-expiration";
import { UpdateKeyMetadata } from "./update-key-metadata";
import { UpdateKeyRatelimit } from "./update-key-ratelimit";
import { UpdateKeyRemaining } from "./update-key-remaining";
import { UpdateKeyName } from "./update-key-name";
export const revalidate = 0;

type Props = {
  params: {
    keyId: string;
  };
};

export default async function SettingsPage(props: Props) {
  const tenantId = getTenantId();

  const key = await db.query.keys.findFirst({
    where: eq(schema.keys.id, props.params.keyId),
    with: {
      workspace: true,
    },
  });
  if (!key || key.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div className="flex flex-col gap-8 mb-20 ">
      <UpdateKeyRemaining apiKey={key} />
      <UpdateKeyRatelimit apiKey={key} />
      <UpdateKeyExpiration apiKey={key} />
      <UpdateKeyMetadata apiKey={key} />
      <UpdateKeyName apiKey={key} />
      <Card>
        <CardHeader>
          <CardTitle>Key ID</CardTitle>
          <CardDescription>This is your key id. It's used in some API calls.</CardDescription>
        </CardHeader>
        <CardContent>
          <Code className="flex items-center justify-between w-full h-8 max-w-sm gap-4">
            <pre>{key.id}</pre>
            <div className="flex items-start justify-between gap-4">
              <CopyButton value={key.id} />
            </div>
          </Code>
        </CardContent>
      </Card>
      <DeleteKey apiKey={key} />
    </div>
  );
}
