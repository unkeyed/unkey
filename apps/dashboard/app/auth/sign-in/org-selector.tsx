"use client";

import { DialogContainer } from "@/components/dialog-container";

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
    "unkey_last_org_name",
    undefined,
  );
  // Set client ready after hydration
  useEffect(() => {
    setClientReady(true);
    // Only open the dialog after hydration to prevent hydration mismatch
    setIsOpen(true);
  }, []);

  const submit = async (orgId: string, orgName: string) => {
    if (isLoading) {
      return;
    }
    setIsLoading(orgId);
    await completeOrgSelection(orgId);
    setLastUsed(orgName);
    setIsLoading(null);
    setIsOpen(false);
  };

  return (
    <DialogContainer
      className="dark"
      isOpen={clientReady && isOpen}
      onOpenChange={(open) => {
        setIsOpen(open);
      }}
      title="Select a workspace"
    >
      <ul className="flex flex-col gap-4 w-full overflow-y-auto max-h-96">
        {organizations
          .sort((a, b) => a.name.localeCompare(b.name))
          .map((org) => (
            <Button
              variant={lastUsed === org.name ? "primary" : "outline"}
              size="2xlg"
              loading={isLoading === org.id}
              key={org.id}
              onClick={() => submit(org.id, org.name)}
            >
              {org.name}
              {lastUsed === org.name ? (
                <span className="absolute right-4 text-xs text-content-subtle">Last used</span>
              ) : null}
            </Button>
          ))}
      </ul>
    </DialogContainer>
  );
};
