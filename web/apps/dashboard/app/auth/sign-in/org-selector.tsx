"use client";

import type { Organization } from "@/lib/auth/types";
import { AuthErrorCode } from "@/lib/auth/types";
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
import { useRouter } from "next/navigation";
import type React from "react";
import { useCallback, useContext, useMemo, useState } from "react";
import { completeOrgSelection } from "../actions";
import { SignInContext } from "../context/signin-context";

interface OrgSelectorProps {
  organizations: Organization[];
  lastOrgId?: string;
}

export const OrgSelector: React.FC<OrgSelectorProps> = ({ organizations, lastOrgId }) => {
  const context = useContext(SignInContext);
  if (!context) {
    throw new Error("OrgSelector must be used within SignInProvider");
  }
  const { setError } = context;
  const router = useRouter();

  const sortedOrgs = useMemo(() => {
    // Sort: recently created first (as proxy for recently used until we track that)
    return [...organizations].sort((a, b) => {
      const aDate = a.createdAt ? new Date(a.createdAt).getTime() : 0;
      const bDate = b.createdAt ? new Date(b.createdAt).getTime() : 0;
      return bDate - aDate; // Newest first
    });
  }, [organizations]);

  // Initialize state directly - no effect needed
  const initialOrgId =
    lastOrgId && sortedOrgs.some((org) => org.id === lastOrgId)
      ? lastOrgId
      : sortedOrgs[0]?.id || "";

  const [isOpen] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [selectedOrgId, setSelectedOrgId] = useState(initialOrgId);

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

          // If session expired, redirect to sign-in to clear stale state
          if (result.code === AuthErrorCode.PENDING_SESSION_EXPIRED) {
            router.push("/auth/sign-in");
          }

          return false;
        }

        // On success, redirect to the dashboard
        router.push(result.redirectTo);
        return true;
      } catch (error) {
        const errorMessage =
          error instanceof Error
            ? error.message
            : "Failed to complete organization selection. Please re-authenticate or contact support@unkey.com";
        toast.error(
          "Failed to complete organization selection. Please re-authenticate or contact support@unkey.com",
        );
        setError(errorMessage);
        setIsLoading(false);
        return false;
      }
    },
    [isLoading, setError, router],
  );

  const handleSubmit = useCallback(async () => {
    await submit(selectedOrgId);
  }, [submit, selectedOrgId]);

  return (
    <DialogContainer
      className="dark bg-black [&_button[aria-label*='Close']]:hidden"
      isOpen={isOpen}
      onOpenChange={() => {
        // Prevent closing the modal - user must select an org
      }}
      preventOutsideClose={true}
      title="Select your workspace"
      footer={
        <div className="flex items-center justify-center text-sm w-full text-content-subtle">
          Select a workspace to sign in.
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
                You don&apos;t have access to any workspaces. Please contact your administrator or
                create a new workspace.
              </p>
              <div className="flex flex-col gap-2 w-full max-w-sm">
                <Button
                  onClick={() => {
                    window.location.href = "mailto:support@unkey.com";
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
