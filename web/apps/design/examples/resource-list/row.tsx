import {
  Button,
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  ResourceListRow,
} from "@unkey/ui";
import { useState } from "react";

const environments = [
  { id: "env_prod", name: "production", domain: "acme-api.unkey.app" },
  { id: "env_preview", name: "preview", domain: "preview.acme-api.unkey.app" },
];

export default function ResourceListRows() {
  const [opened, setOpened] = useState<string | null>(null);

  return (
    <ResourceList>
      <ResourceListContent>
        <ResourceListBody>
          {environments.map((env) => (
            <ResourceListItem key={env.id}>
              <ResourceListRow
                label={`Open ${env.name}`}
                onActivate={() => setOpened(env.id)}
                expanded={opened === env.id}
                className="flex items-center gap-4"
              >
                <span className="min-w-0 flex-1 truncate font-medium text-gray-12 text-sm">
                  {env.name}
                </span>
                <span className="font-mono text-gray-11 text-xs">{env.domain}</span>
                <Button variant="outline" size="sm" className="relative">
                  Copy
                </Button>
              </ResourceListRow>
            </ResourceListItem>
          ))}
        </ResourceListBody>
      </ResourceListContent>
    </ResourceList>
  );
}
