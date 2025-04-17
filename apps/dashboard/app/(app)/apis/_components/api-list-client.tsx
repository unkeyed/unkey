"use client";

import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { useState } from "react";
import { ApiListGrid } from "./api-list-grid";
import { ApiListControlCloud } from "./control-cloud";
import { ApiListControls } from "./controls";

export const ApiListClient = () => {
  const [isSearching, setIsSearching] = useState<boolean>(false);
  const [apiList, setApiList] = useState<ApiOverview[]>([]);

  return (
    <div className="flex flex-col">
      <ApiListControls apiList={apiList} onApiListChange={setApiList} onSearch={setIsSearching} />
      <ApiListControlCloud />
      <ApiListGrid isSearching={isSearching} setApiList={setApiList} apiList={apiList} />
    </div>
  );
};
