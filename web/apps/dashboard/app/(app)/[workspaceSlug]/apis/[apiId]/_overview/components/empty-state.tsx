"use client";

import { trpc } from "@/lib/trpc/client";
import { BookBookmark, Plus } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { CreateKeyDialog } from "../../_components/create-key";

export function ApiRequestsEmptyState({ apiId }: { apiId: string }) {
  const { data: layoutData } = trpc.api.queryApiKeyDetails.useQuery(
    { apiId },
    { enabled: Boolean(apiId) },
  );

  const keyspaceId = layoutData?.keyAuth?.id ?? null;
  const keyspaceDefaults = layoutData?.currentApi.keyspaceDefaults ?? null;

  return (
    <div className="flex-1 min-h-[300px] w-full flex items-center justify-center">
      <Empty className="w-[400px] flex items-start">
        <Empty.Icon className="w-auto" />
        <Empty.Title>No verification data yet</Empty.Title>
        <Empty.Description className="text-left">
          This API hasn't verified any keys yet. Create a key, use it in a request, and verification
          activity will show up here.
        </Empty.Description>
        <Empty.Actions className="mt-4 justify-center md:justify-start">
          {/* CreateKeyDialog also renders its (invisible) form and dialog as
              siblings — wrap it so only one flex item lands in the gap-4 row. */}
          <div className="flex items-center">
            <CreateKeyDialog
              keyspaceId={keyspaceId}
              apiId={apiId}
              keyspaceDefaults={keyspaceDefaults}
              trigger={(open) => (
                <Button onClick={open} size="md">
                  <Plus />
                  Create key
                </Button>
              )}
            />
          </div>
          <a
            href="https://www.unkey.com/docs/introduction"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Button size="md" variant="outline">
              <BookBookmark />
              Documentation
            </Button>
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  );
}
