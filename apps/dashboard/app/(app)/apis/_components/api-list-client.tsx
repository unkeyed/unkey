"use client";
import { Button, Empty } from "@unkey/ui";
import { BookOpen } from "lucide-react";
import Link from "next/link";
import type { API } from "../page";
import { ApiListCard } from "./api-list-card";
import { ApiListControlCloud } from "./control-cloud";
import { ApiListControls } from "./controls";
import { CreateApiButton } from "./create-api-button";

export const ApiListClient = ({ apiList }: { apiList: API[] }) => {
  return (
    <div className="flex flex-col">
      <ApiListControls />
      <ApiListControlCloud />

      <div className="p-5">
        {apiList.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5 w-full max-w-7xl">
            {apiList.map((api) => (
              <ApiListCard api={api} key={api.id} />
            ))}
          </div>
        ) : (
          <Empty>
            <Empty.Icon />
            <Empty.Title>No APIs found</Empty.Title>
            <Empty.Description>
              You haven&apos;t created any APIs yet. Create one to get started.
            </Empty.Description>
            <Empty.Actions>
              <CreateApiButton key="createApi" />
              <Link href="/docs" target="_blank">
                <Button>
                  <BookOpen />
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
