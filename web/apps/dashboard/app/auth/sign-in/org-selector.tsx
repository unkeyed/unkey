"use client";

import type { Organization } from "@/lib/auth/types";
import {
  Button,
  DialogContainer,
  Empty,
  Loading,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import type React from "react";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { completeOrgSelection } from "../actions";
import { SignInContext } from "../context/signin-context";

interface OrgSelectorProps {
  organizations: Organization[];
  lastOrgId?: string;
  onClose?: () => void;
}

export const OrgSelector: React.FC<OrgSelectorProps> = ({ organizations, lastOrgId, onClose }) => {
  const context = useContext(SignInContext);
  if (!context) {
    throw new Error("OrgSelector must be used within SignInProvider");
  }
  const { setError } = context;
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [clientReady, setClientReady] = useState(false);
  const [selectedOrgId, setSelectedOrgId] = useState("");
  const [hasInitialized, setHasInitialized] = useState(false);

  // Clear error when closing explicitly (Cancel button or X button)
  // This prevents showing stale errors when switching accounts
  const handleClose = useCallback(() => {
    setError(null);
    setIsOpen(false);
    onClose?.();
  }, [setError, onClose]);
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
          toast.error(result.message);
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
        toast.error(
          "Failed to complete organization selection. Please re-authenticate or contact support@unkey.dev",
        );
        setError(errorMessage);
        setIsLoading(false);
        return false;
      }
    },
    [isLoading, setError],
  );

  const handleSubmit = useCallback(async () => {
    await submit(selectedOrgId);
  }, [submit, selectedOrgId]);

  // Initialize org selector when client is ready
  useEffect(() => {
    if (!clientReady || hasInitialized) {
      return;
    }

    // Pre-select the last used org if it exists in the list, otherwise first org
    const preselectedOrgId =
      lastOrgId && sortedOrgs.some((org) => org.id === lastOrgId)
        ? lastOrgId
        : sortedOrgs[0]?.id || "";

    setSelectedOrgId(preselectedOrgId);
    setIsOpen(true); // Always show the modal for manual selection
    setHasInitialized(true);
  }, [clientReady, sortedOrgs, lastOrgId, hasInitialized]);

  return (
    <DialogContainer
      className="dark bg-black"
      isOpen={clientReady && isOpen}
      onOpenChange={(open) => {
        if (!isLoading && open) {
          // Only allow opening via onOpenChange, not closing via backdrop
          setIsOpen(true);
        }
        // When closing (X button), clear error state
        if (!isLoading && !open) {
          setError(null);
        }
      }}
      title="Select your workspace"
      footer={
        <div className="flex items-center justify-between text-sm w-full">
          <Button variant="ghost" onClick={handleClose}>
            Cancel
          </Button>
          <div className="text-content-subtle">Select a workspace to sign in.</div>
        </div>
      }
    >
      <div className="flex flex-col gap-6 w-full">
        {/* Workspace selector */}
        {sortedOrgs.length === 0 ? (
          <Empty>
            <div className="flex flex-col items-center gap-4 text-center">
              <h3 className="text-lg font-medium text-content">No workspaces found</h3>
              <p className="text-sm text-content-subtle max-w-md">
                You don't have access to any workspaces. Please contact your administrator or create
                a new workspace.
              </p>
              <div className="flex flex-col gap-2 w-full max-w-sm">
                <Button
                  onClick={() => {
                    window.location.href = "mailto:support@unkey.dev";
                  }}
                  className="w-full"
                  size="lg"
                >
                  Contact Support
                </Button>
                <Button
                  onClick={() => {
                    window.location.href = "/auth/sign-out";
                  }}
                  variant="outline"
                  className="w-full"
                  size="lg"
                >
                  Sign Out
                </Button>
              </div>
            </div>
          </Empty>
        ) : (
          <>
            <div className="flex flex-col gap-4">
              <label htmlFor="workspace-selector" className="text-sm font-medium text-content">
                Workspace
              </label>
              <Select value={selectedOrgId} onValueChange={setSelectedOrgId} disabled={isLoading}>
                <SelectTrigger id="workspace-selector">
                  <SelectValue placeholder="Select a workspace..." />
                </SelectTrigger>
                <SelectContent className="dark overflow-y-auto max-h-[400px]">
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
                  <Loading type="spinner" />
                  <span>Signing in...</span>
                </div>
              ) : (
                "Continue"
              )}
            </Button>
          </>
        )}
      </div>
    </DialogContainer>
  );
};
