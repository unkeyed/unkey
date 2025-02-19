"use client";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Button, Empty } from "@unkey/ui";
import { BookOpen, Code, Link } from "lucide-react";
import { type PropsWithChildren, useState } from "react";
import { RatelimitListControlCloud } from "./control-cloud";
import { RatelimitListControls } from "./controls";
import { NamespaceCard } from "./namespace-card";

export const RatelimitClient = ({
  ratelimitNamespaces,
}: PropsWithChildren<{
  ratelimitNamespaces: {
    id: string;
    name: string;
  }[];
}>) => {
  const [namespaces, setNamespaces] = useState(ratelimitNamespaces);
  const snippet = `curl -XPOST 'https://api.unkey.dev/v1/ratelimits.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -d '{
      "namespace": "demo_namespace",
      "identifier": "user_123",
      "limit": 10,
      "duration": 10000
  }'`;

  return (
    <div className="flex flex-col">
      <RatelimitListControls
        setNamespaces={setNamespaces}
        initialNamespaces={ratelimitNamespaces}
      />
      <RatelimitListControlCloud />
      <div className="p-5">
        {" "}
        {namespaces.length > 0 ? (
          <div className="grid grid-cols-3 gap-5 w-full max-w-7xl">
            {namespaces.map((rn) => (
              <NamespaceCard namespace={rn} key={rn.id} />
            ))}
          </div>
        ) : (
          <Empty>
            <Empty.Icon />
            <Empty.Title>No Namespaces found</Empty.Title>
            <Empty.Description>
              You haven&apos;t created any Namespaces yet. Create one by performing a limit request
              as shown below.
            </Empty.Description>
            <Code className="flex items-start gap-8 p-4 my-8 text-xs text-left">
              {snippet}
              <CopyButton value={snippet} />
            </Code>
            <Empty.Actions>
              <Link href="/docs/ratelimiting/introduction" target="_blank">
                <Button className="items-center w-full gap-2 ">
                  <BookOpen className="w-4 h-4 " />
                  Read the docs
                </Button>
              </Link>
            </Empty.Actions>
          </Empty>
        )}
      </div>
    </div>
  );
};
