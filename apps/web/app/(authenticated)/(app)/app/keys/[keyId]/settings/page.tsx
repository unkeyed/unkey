import { CopyButton } from "@/components/dashboard/copy-button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { notFound } from "next/navigation";
import { DeleteKey } from "./delete-key";
import { UpdateKeyEnabled } from "./update-key-enabled";
import { UpdateKeyExpiration } from "./update-key-expiration";
import { UpdateKeyMetadata } from "./update-key-metadata";
import { UpdateKeyName } from "./update-key-name";
import { UpdateKeyOwnerId } from "./update-key-owner-id";
import { UpdateKeyRatelimit } from "./update-key-ratelimit";
import { UpdateKeyRemaining } from "./update-key-remaining";

type Props = {
  params: {
    keyId: string;
  };
};

export default async function SettingsPage(props: Props) {
  const tenantId = getTenantId();

  const key = await db.query.keys.findFirst({
    where: and(eq(schema.keys.id, props.params.keyId), isNull(schema.keys.deletedAt)),

    with: {
      workspace: true,
    },
  });
  if (!key || key.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div className="mb-20 flex flex-col gap-8 ">
      <UpdateKeyEnabled apiKey={key} />
      <UpdateKeyRemaining apiKey={key} />
      <UpdateKeyRatelimit apiKey={key} />
      <UpdateKeyExpiration apiKey={key} />
      <UpdateKeyMetadata apiKey={key} />
      <UpdateKeyName apiKey={key} />
      <UpdateKeyOwnerId apiKey={key} />
      <Card>
        <CardHeader>
          <CardTitle>Key ID</CardTitle>
          <CardDescription>This is your key id. It's used in some API calls.</CardDescription>
        </CardHeader>
        <CardContent>
          <Code className="flex h-8 w-full max-w-sm items-center justify-between gap-4">
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
