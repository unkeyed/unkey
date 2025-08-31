"use client";

import { Button } from "@unkey/ui";
import { useState } from "react";
import { ApiListClient } from "./api-list-client";
import { ApiListClientTanStack } from "./api-list-client-tanstack";

export const ApiListWrapper = () => {
  const [useTanStack, setUseTanStack] = useState(false);

  return (
    <div>
      <div className="flex justify-end p-5 pb-0">
        <Button
          onClick={() => setUseTanStack(!useTanStack)}
          variant="outline"
          size="sm"
        >
          {useTanStack ? "Switch to tRPC" : "Switch to TanStack DB"}
        </Button>
      </div>
      
      {useTanStack ? <ApiListClientTanStack /> : <ApiListClient />}
    </div>
  );
};