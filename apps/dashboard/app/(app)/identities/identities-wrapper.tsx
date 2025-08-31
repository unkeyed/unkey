"use client";

import { Button } from "@unkey/ui";
import { useState } from "react";
import { IdentitiesTanStack } from "./identities-tanstack";

export function IdentitiesWrapper({ children }: { children: React.ReactNode }) {
  const [useTanStack, setUseTanStack] = useState(false);

  if (useTanStack) {
    return (
      <>
        <div className="fixed top-4 right-4 z-50">
          <Button variant="outline" size="sm" onClick={() => setUseTanStack(false)}>
            Switch to Server-Side
          </Button>
        </div>
        <IdentitiesTanStack />
      </>
    );
  }

  return (
    <>
      <div className="fixed top-4 right-4 z-50">
        <Button variant="outline" size="sm" onClick={() => setUseTanStack(true)}>
          Switch to TanStack DB
        </Button>
      </div>
      {children}
    </>
  );
}
