"use client";
import { EmptyComponentSpacer } from "@/components/empty-component-spacer";
import { Button, CopyButton, Empty } from "@unkey/ui";
import { BookOpen } from "lucide-react";
import { type PropsWithChildren, useState } from "react";
import { RatelimitListControlCloud } from "./control-cloud";
import { RatelimitListControls } from "./controls";
import { NamespaceCard } from "./namespace-card";

const EXAMPLE_SNIPPET = `curl -XPOST 'https://api.unkey.dev/v1/ratelimits.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -d '{
      "namespace": "demo_namespace",
      "identifier": "user_123",
      "limit": 10,
      "duration": 10000
  }'`;

export const RatelimitClient = ({
  ratelimitNamespaces,
}: PropsWithChildren<{
  ratelimitNamespaces: {
    id: string;
    name: string;
  }[];
}>) => {
  const [namespaces, setNamespaces] = useState(ratelimitNamespaces);

  return (
    <div className="flex flex-col">
      <RatelimitListControls
        setNamespaces={setNamespaces}
        initialNamespaces={ratelimitNamespaces}
      />
      <RatelimitListControlCloud />

      {namespaces.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5 w-full p-5">
          {namespaces.map((namespace) => (
            <NamespaceCard namespace={namespace} key={namespace.id} />
          ))}
        </div>
      ) : (
        <EmptyComponentSpacer>
          <Empty className="max-w-2xl mx-auto">
            <Empty.Icon />
            <Empty.Title>No Namespaces found</Empty.Title>
            <Empty.Description>
              You haven't created any Namespaces yet. Create one by performing a limit request as
              shown below.
            </Empty.Description>

            <div className="w-full mt-8 mb-8">
              <div className="flex items-start gap-4 p-4 bg-gray-2 border border-gray-6 rounded-lg">
                <pre className="flex-1 text-xs text-left overflow-x-auto">
                  <code>{EXAMPLE_SNIPPET}</code>
                </pre>
                <CopyButton value={EXAMPLE_SNIPPET} />
              </div>
            </div>

            <Empty.Actions>
              <a href="/docs/ratelimiting/introduction" target="_blank" rel="noopener noreferrer">
                <Button className="flex items-center w-full gap-2">
                  <BookOpen className="w-4 h-4" />
                  Read the docs
                </Button>
              </a>
            </Empty.Actions>
          </Empty>
        </EmptyComponentSpacer>
      )}
    </div>
  );
};
