"use client";

import {
  Empty,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import type { Organization } from "@/lib/auth/types";
import { Button, DialogContainer, Loading } from "@unkey/ui";
import type React from "react";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { completeOrgSelection } from "../actions";
import { SignInContext } from "../context/signin-context";

interface OrgSelectorProps {
  organizations: Organization[];
  lastOrgId?: string;
}

export const OrgSelector: React.FC<OrgSelectorProps> = ({
  organizations,
  lastOrgId,
}) => {
  const context = useContext(SignInContext);
  if (!context) {
    throw new Error("OrgSelector must be used within SignInProvider");
  }
  const { setError } = context;
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isAttemptingAutoSignIn, setIsAttemptingAutoSignIn] = useState(false);
  const [clientReady, setClientReady] = useState(false);
  const [selectedOrgId, setSelectedOrgId] = useState("");
  const [hasAttemptedAutoSelection, setHasAttemptedAutoSelection] =
    useState(false);
  // Set client ready after hydration
  useEffect(() => {
    setClientReady(true);
  }, []);

  const sortedOrgs = useMemo(() => {
    // Sort: recently created first (as proxy for recently used until we track that)
    return [...organizations].sort((a, b) => {
      const aDate = a.createdAt ? new Date(a.createdAt).getTime() : 0;
      const bDate = b.createdAt ? new Date(b.createdAt).getTime() : 0;
      return bDate - aDate; // Newest first
    });
  }, [organizations]);

  const submit = useCallback(
    async (orgId: string): Promise<boolean> => {
      if (isLoading || !orgId) {
        return false;
      }

      try {
        setIsLoading(true);
        const result = await completeOrgSelection(orgId);

        if (!result.success) {
          setError(result.message);
          setIsLoading(false);
          return false;
        }
        // On success, redirect to the dashboard
        window.location.href = result.redirectTo;
        return true;
      } catch (error) {
        const errorMessage =
          error instanceof Error
            ? error.message
            : "Failed to complete organization selection. Please re-authenticate or contact support@unkey.dev";

        setError(errorMessage);
        setIsLoading(false);
        return false;
      }
    },
    [isLoading]
  );

  const handleSubmit = useCallback(() => {
    submit(selectedOrgId);
  }, [submit, selectedOrgId]);

  // Auto-select last used organization if available
  useEffect(() => {
    if (!clientReady) {
      return;
    }

    setHasAttemptedAutoSelection(true);

    if (lastOrgId) {
      // Check if the stored orgId exists in the current list of organizations
      const orgExists = sortedOrgs.some((org) => org.id === lastOrgId);

      if (orgExists) {
        // Show loading state while attempting auto sign-in
        setIsAttemptingAutoSignIn(true);
        // Auto-submit this organization and handle failure by reopening the dialog
        submit(lastOrgId).then((success) => {
          if (!success) {
            // If auto-submit fails, pre-select the last used org and open the dialog for manual selection
            setSelectedOrgId(lastOrgId);
            setIsOpen(true);
          }
        });
        return;
      }
    }

    // If no auto-selection, show the modal with first org pre-selected (from sorted array)
    // Use sortedOrgs[0].id to ensure the pre-selected value matches the displayed first option
    if (sortedOrgs.length > 0 && sortedOrgs[0]?.id) {
      setSelectedOrgId(sortedOrgs[0].id);
    }
  }, [clientReady, sortedOrgs, hasAttemptedAutoSelection, submit, lastOrgId]);

  return (
    !isAttemptingAutoSignIn && (
      <DialogContainer
        className="dark bg-black"
        isOpen={clientReady && isOpen}
        onOpenChange={(open) => {
          setIsOpen(open);
        }}
        title={!isAttemptingAutoSignIn ? "Select your workspace" : ""}
        footer={
          !isAttemptingAutoSignIn && (
            <div className="flex items-center justify-center text-sm w-full text-content-subtle">
              Select a workspace to sign in.
            </div>
          )
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
            <Select
              value={selectedOrgId}
              onValueChange={setSelectedOrgId}
              disabled={isLoading}
            >
              <SelectTrigger id="workspace-selector">
                <SelectValue placeholder="Select a workspace..." />
              </SelectTrigger>
              <SelectContent>
                {sortedOrgs.map((org) => (
                  <SelectItem key={org.id} value={org.id}>
                    {org.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
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
                <Loading type="spinner" size={100} />
                <span>Signing in...</span>
              </div>
            ) : (
              "Continue"
            )}
          </Button>
        </div>
      </DialogContainer>
    )
  );
};
