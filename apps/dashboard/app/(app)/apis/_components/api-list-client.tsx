"use client";

import type { ApisOverviewResponse } from "@/lib/trpc/routers/api/overview/schemas";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { ApiListGrid } from "./api-list-grid";
import { ApiListControlCloud } from "./control-cloud";
import { ApiListControls } from "./controls";
import { CreateApiButton } from "./create-api-button";

export const ApiListClient = ({
  initialData,
}: {
  initialData: ApisOverviewResponse;
}) => {
  return (
    <div className="flex flex-col">
      <ApiListControls />
      <ApiListControlCloud />
      {initialData.apiList.length > 0 ? (
        <div className="p-5">
          <ApiListGrid initialData={initialData} />
        </div>
      ) : (
        <div className="h-screen flex items-center justify-center -translate-y-32">
          <div className="flex justify-center items-center">
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
                  <Button>
                    <BookBookmark />
                    Documentation
                  </Button>
                </a>
              </Empty.Actions>
            </Empty>
          </div>
        </div>
      )}
    </div>
  );
};
