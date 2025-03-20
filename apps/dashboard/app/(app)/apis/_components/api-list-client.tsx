"use client";

import { EmptyComponentSpacer } from "@/components/empty-component-spacer";
import type {
  ApiOverview,
  ApisOverviewResponse,
} from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useState } from "react";
import { ApiListGrid } from "./api-list-grid";
import { ApiListControlCloud } from "./control-cloud";
import { ApiListControls } from "./controls";
import { CreateApiButton } from "./create-api-button";

export const ApiListClient = ({
  initialData,
}: {
  initialData: ApisOverviewResponse;
}) => {
  const [isSearching, setIsSearching] = useState<boolean>(false);
  const [apiList, setApiList] = useState<ApiOverview[]>(initialData.apiList);

  return (
    <div className="flex flex-col">
      <ApiListControls apiList={apiList} onApiListChange={setApiList} onSearch={setIsSearching} />
      <ApiListControlCloud />
      {initialData.apiList.length > 0 ? (
        <ApiListGrid
          isSearching={isSearching}
          initialData={initialData}
          setApiList={setApiList}
          apiList={apiList}
        />
      ) : (
        <EmptyComponentSpacer>
          <Empty className="m-0 p-0">
            <Empty.Icon />
            <Empty.Title>No APIs found</Empty.Title>
            <Empty.Description>
              You haven&apos;t created any APIs yet. Create one to get started.
            </Empty.Description>
            <Empty.Actions className="mt-4 ">
              <CreateApiButton />
              <a
                href="https://www.unkey.com/docs/introduction"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button size="md">
                  <BookBookmark />
                  Documentation
                </Button>
              </a>
            </Empty.Actions>
          </Empty>
        </EmptyComponentSpacer>
      )}
    </div>
  );
};
