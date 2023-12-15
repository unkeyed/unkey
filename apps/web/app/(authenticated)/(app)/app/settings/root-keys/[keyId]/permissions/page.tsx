import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Code } from "@/components/ui/code";
import { Label } from "@/components/ui/label";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";

import { notFound } from "next/navigation";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RootKeyPage(props: {
  params: { keyId: string };
  searchParams: {
    interval?: Interval;
  };
}) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: {
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });
  if (!workspace) {
    return notFound();
  }

  const key = await db.query.keys.findFirst({
    where: eq(schema.keys.forWorkspaceId, workspace.id) && eq(schema.keys.id, props.params.keyId),
    with: {
      roles: {
        with: {
          role: {
            columns: {
              name: true,
            },
          },
        },
      },
    },
  });
  if (!key) {
    return notFound();
  }

  type Root = {
    children: Record<string, Node>;
  };
  type Node = {
    role: string;
    path: string[];
    children: Record<string, Node>;
  };

  const roleNames = key.roles.map((r) => r.role.name);

  type WorkspaceId = string;
  type ApiId = string;
  type Action = string;
  type UnkeyRoleName =
    | `${WorkspaceId}::apis:${ApiId}::keys:${Action}`
    | `${WorkspaceId}::apis:${ApiId}::${Action}`
    | `${WorkspaceId}::${Action}`;

  type UnkeyPermissions = {
    [workspaceId: string]: {
      actions: {
        createApi?: boolean;
      };
      apis: {
        [apiId: string]: {
          actions: {
            read?: boolean;
            update?: boolean;
            createKey?: boolean;
          };
          keys: {
            [keyId: string]: {
              actions: {
                read?: boolean;
                update?: boolean;
                delete?: boolean;
              };
            };
          };
        };
      };
    };
  };

  return (
    <div className="">
      <Card>
        <CardHeader>
          <CardTitle>Permissions</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-1">
          {roleNames.map((role) => (
            <Badge variant="secondary" key={role}>
              {" "}
              {role}
            </Badge>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
