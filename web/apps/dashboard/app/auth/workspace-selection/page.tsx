"use client";

import type { Organization } from "@/lib/auth/types";
import {
  Button,
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
import { completeOrgSelection } from "../sign-in/actions";
import { SignInContext } from "../sign-in/context/signin-context";

interface WorkspaceSelectionPageProps {
  organizations: Organization[];
  lastOrgId?: string;
}

export default function WorkspaceSelectionPage({
  organizations,
  lastOrgId,
}: WorkspaceSelectionPageProps) {
  const context = useContext(SignInContext);
  if (!context) {
    throw new Error("WorkspaceSelectionPage must be used within SignInProvider");
  }
  const { setError } = context;
  const [isLoading, setIsLoading] = useState(false);
  const [clientReady, setClientReady] = useState(false);
  const [selectedOrgId, setSelectedOrgId] = useState("");

  // Clear error when mounting
  useEffect(() => {
    setError(null);
  }, [setError]);

  // Set client ready after hydration
  useEffect(() => {
    setClientReady(true);
  }, []);

  const sortedOrgs = useMemo(() => {
    return [...organizations].sort((a, b) => {
      const aDate = a.createdAt ? new Date(a.createdAt).getTime() : 0;
      const bDate = b.createdAt ? new Date(b.createdAt).getTime() : 0;
      return bDate - aDate; // Newest first
    });
  }, [organizations]);

  // Pre-select the last used org if it exists in the list, otherwise first org
  useEffect(() => {
    if (!clientReady) return;

    const preselectedOrgId =
      lastOrgId && sortedOrgs.some((org) => org.id === lastOrgId)
        ? lastOrgId
        : sortedOrgs[0]?.id || "";

    setSelectedOrgId(preselectedOrgId);
  }, [clientReady, sortedOrgs, lastOrgId]);

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

  if (!clientReady) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-black">
        <Loading type="spinner" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-black flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="bg-gray-1 dark:bg-black border border-gray-4 rounded-2xl p-6 md:p-8">
          <h1 className="text-xl font-semibold text-content mb-2">Select your workspace</h1>
          <p className="text-sm text-content-subtle mb-6">
            Choose a workspace to continue to your dashboard.
          </p>

          {sortedOrgs.length === 0 ? (
            <Empty>
              <div className="flex flex-col items-center gap-4 text-center">
                <h3 className="text-lg font-medium text-content">No workspaces found</h3>
                <p className="text-sm text-content-subtle max-w-md">
                  You don't have access to any workspaces. Please contact your administrator.
                </p>
                <Button
                  onClick={() => {
                    window.location.href = "mailto:support@unkey.dev";
                  }}
                  className="w-full"
                  size="lg"
                >
                  Contact Support
                </Button>
              </div>
            </Empty>
          ) : (
            <div className="flex flex-col gap-6">
              <div className="flex flex-col gap-2">
                <label htmlFor="workspace-selector" className="text-sm font-medium text-content">
                  Workspace
                </label>
                <Select value={selectedOrgId} onValueChange={setSelectedOrgId} disabled={isLoading}>
                  <SelectTrigger id="workspace-selector">
                    <SelectValue placeholder="Select a workspace..." />
                  </SelectTrigger>
                  <SelectContent className="overflow-y-auto max-h-[400px]">
                    {sortedOrgs.map((org) => (
                      <SelectItem key={org.id} value={org.id}>
                        {org.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

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
            </div>
          )}
        </div>
      </div>
    </div>
  );
}