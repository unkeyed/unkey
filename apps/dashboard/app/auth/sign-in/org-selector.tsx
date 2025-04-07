"use client";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import type { Organization } from "@/lib/auth/types";
import { Button } from "@unkey/ui";
import type React from "react";
import { useEffect, useState } from "react";
import { useLocalStorage } from "usehooks-ts";
import { completeOrgSelection } from "../actions";

interface OrgSelectorProps {
  organizations: Organization[];
}

export const OrgSelector: React.FC<OrgSelectorProps> = ({ organizations }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState<null | string>(null);
  const [clientReady, setClientReady] = useState(false);
  const [lastUsed, setLastUsed] = useLocalStorage<string | undefined>(
    "unkey_last_org_id",
    undefined,
  );
  // Set client ready after hydration
  useEffect(() => {
    setClientReady(true);
    // Only open the dialog after hydration to prevent hydration mismatch
    setIsOpen(true);
  }, []);

  const submit = async (orgId: string) => {
    if (isLoading) {
      return;
    }
    setIsLoading(orgId);
    await completeOrgSelection(orgId);
    setLastUsed(orgId);
    setIsLoading(null);
    setIsOpen(false);
  };

  return (
    <Dialog
      open={clientReady && isOpen}
      onOpenChange={(open) => {
        setIsOpen(open);
      }}
    >
      <DialogContent className="dark border-border w-11/12">
        <DialogHeader className="dark">
          <DialogTitle className="text-white">Workspace Selection</DialogTitle>
          <DialogDescription className="dark">
            Select a workspace to continue authentication:
          </DialogDescription>
        </DialogHeader>

        <ul className="flex flex-col gap-4 w-full overflow-y-auto max-h-96">
          {organizations
            .sort((a, b) => a.name.localeCompare(b.name))
            .map((org) => (
              <Button
                variant={lastUsed === org.id ? "primary" : "outline"}
                size="2xlg"
                loading={isLoading === org.id}
                key={org.id}
                onClick={() => submit(org.id)}
              >
                {org.name}
                {lastUsed === org.id ? (
                  <span className="absolute right-4 text-xs text-content-subtle">Last used</span>
                ) : null}
              </Button>
            ))}
        </ul>
      </DialogContent>
    </Dialog>
  );
};
