"use client";

import { DialogContainer } from "@unkey/ui";

import type { Organization } from "@/lib/auth/types";
import { Button } from "@unkey/ui";
import type React from "react";
import { useEffect, useState } from "react";
import { completeOrgSelection } from "../actions";

interface OrgSelectorProps {
  organizations: Organization[];
  onError: (errorMessage: string) => void;
}

export const OrgSelector: React.FC<OrgSelectorProps> = ({ organizations, onError }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState<null | string>(null);
  const [clientReady, setClientReady] = useState(false);

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

    try {
      setIsLoading(orgId);
      const result = await completeOrgSelection(orgId);

      if (!result.success) {
        onError(result.message);
      }

      return;
    } catch (error) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : "Failed to complete organization selection. Please re-authenticate or contact support@unkey.dev";

      onError(errorMessage);
    } finally {
      setIsLoading(null);
      setIsOpen(false);
    }
  };

  return (
    <DialogContainer
      className="dark bg-black"
      isOpen={clientReady && isOpen}
      onOpenChange={(open) => {
        setIsOpen(open);
      }}
      title="Select a workspace"
      footer={
        <div className="flex items-center justify-center text-sm w-full">
          Select a workspace to sign in.
        </div>
      }
    >
      <ul className="flex flex-col gap-4 w-full overflow-y-auto max-h-96">
        {organizations
          .sort((a, b) => a.name.localeCompare(b.name))
          .map((org) => (
            <Button
              className="dark"
              variant="default"
              size="2xlg"
              loading={isLoading === org.id}
              key={org.id}
              onClick={() => submit(org.id)}
            >
              {org.name}
            </Button>
          ))}
      </ul>
    </DialogContainer>
  );
};
