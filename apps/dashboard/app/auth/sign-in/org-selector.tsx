"use client";

import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import type { Organization } from "@/lib/auth/types";
import { Button, DialogContainer, Loading } from "@unkey/ui";
import type React from "react";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { completeOrgSelection } from "../actions";
import { SignInContext } from "../context/signin-context";

interface OrgSelectorProps {
  organizations: Organization[];
}

export const OrgSelector: React.FC<OrgSelectorProps> = ({ organizations }) => {
  const context = useContext(SignInContext);
  if (!context) {
    throw new Error("OrgSelector must be used within SignInProvider");
  }
  const { setError } = context;
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [clientReady, setClientReady] = useState(false);
  const [selectedOrgId, setSelectedOrgId] = useState("");
  const [hasAttemptedAutoSelection, setHasAttemptedAutoSelection] =
    useState(false);
  // Set client ready after hydration
  useEffect(() => {
    setClientReady(true);
  }, []);

  const orgOptions: ComboboxOption[] = useMemo(() => {
    // Sort: recently created first (as proxy for recently used until we track that)
    const sorted = [...organizations].sort((a, b) => {
      const aDate = a.createdAt ? new Date(a.createdAt).getTime() : 0;
      const bDate = b.createdAt ? new Date(b.createdAt).getTime() : 0;
      return bDate - aDate; // Newest first
    });

    return sorted.map((org) => ({
      label: org.name,
      value: org.id,
      searchValue: org.name.toLowerCase(),
    }));
  }, [organizations]);

  const submit = useCallback(
    async (orgId: string) => {
      if (isLoading || !orgId) {
        return;
      }

      try {
        setIsLoading(true);
        const result = await completeOrgSelection(orgId);

        if (!result.success) {
          setError(result.message);
          setIsLoading(false);
        }
        // On success, the page will redirect, so we don't need to reset loading state
      } catch (error) {
        const errorMessage =
          error instanceof Error
            ? error.message
            : "Failed to complete organization selection. Please re-authenticate or contact support@unkey.dev";

        setError(errorMessage);
        setIsLoading(false);
      }
    },
    [isLoading, setError]
  );

  const handleSubmit = useCallback(() => {
    submit(selectedOrgId);
  }, [submit, selectedOrgId]);

  // Helper function to get cookie value on client side
  const getCookie = (name: string): string | null => {
    if (typeof document === "undefined") {
      return null;
    }
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) {
      return parts.pop()?.split(";").shift() || null;
    }
    return null;
  };

  // Auto-select last used organization if available
  useEffect(() => {
    if (!clientReady || hasAttemptedAutoSelection) {
      return;
    }

    setHasAttemptedAutoSelection(true);

    // Get the last used organization ID from cookie
    const lastUsedOrgId = getCookie("unkey_last_org_used");

    if (lastUsedOrgId) {
      // Check if the stored orgId exists in the current list of organizations
      const orgExists = organizations.some((org) => org.id === lastUsedOrgId);

      if (orgExists) {
        // Auto-submit this organization
        submit(lastUsedOrgId);
        return;
      }
    }

    // If no auto-selection, show the modal with first org pre-selected
    if (organizations.length > 0) {
      setSelectedOrgId(organizations[0].id);
    }
    setIsOpen(true);
  }, [clientReady, organizations, hasAttemptedAutoSelection, submit]);

  return (
    <DialogContainer
      className="dark bg-black"
      isOpen={clientReady && isOpen}
      onOpenChange={(open) => {
        setIsOpen(open);
      }}
      title="Select your workspace"
      footer={
        <div className="flex items-center justify-center text-sm w-full text-content-subtle">
          Select a workspace to sign in.
        </div>
      }
    >
      <div className="flex flex-col gap-6 w-full">
        {/* Workspace selector */}
        <div className="flex flex-col gap-4">
          <label
            htmlFor="workspace-selector"
            className="text-sm font-medium text-content"
          >
            Workspace
          </label>
          <Combobox
            id="workspace-selector"
            options={orgOptions}
            value={selectedOrgId}
            onSelect={setSelectedOrgId}
            placeholder="Select a workspace..."
            searchPlaceholder="Search workspaces..."
            emptyMessage="No workspaces found."
            disabled={isLoading}
          />
        </div>

        {/* Submit button */}
        <Button
          onClick={handleSubmit}
          disabled={isLoading || !selectedOrgId}
          className="w-full"
          size="lg"
        >
          {isLoading ? (
            <div className="flex items-center justify-center gap-2">
              <Loading type="dots" size={20} />
              <span>Signing in...</span>
            </div>
          ) : (
            "Continue"
          )}
        </Button>
      </div>
    </DialogContainer>
  );
};
