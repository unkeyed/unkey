"use client";

import type { Organization } from "@/lib/auth/types";
import { AuthErrorCode, SIGN_IN_URL } from "@/lib/auth/types";
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
import type { Route } from "next";
import { useRouter, useSearchParams } from "next/navigation";
import type React from "react";
import { useCallback, useContext, useMemo, useState } from "react";
import { clearPendingAuth, completeOrgSelection } from "../actions";
import { SignInContext } from "../context/signin-context";
import { consumeRedirectUrl, resolveRedirectUrl } from "./redirect-utils";

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
  const searchParams = useSearchParams();
  const redirectParam = searchParams?.get("redirect");

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

  const [isOpen, setIsOpen] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [selectedOrgId, setSelectedOrgId] = useState(initialOrgId);

  const handleClose = useCallback(async () => {
    // Close modal immediately to prevent flash
    setIsOpen(false);
    // Clear pending auth state and redirect
    await clearPendingAuth();
    router.push("/auth/sign-in" as Route);
  }, [router]);

  const submit = useCallback(
    async (orgId: string): Promise<boolean> => {
      if (isLoading || !orgId) {
        return false;
      }

      try {
        setIsLoading(true);
        const result = await completeOrgSelection(orgId);

        // The selected org requires MFA (or a Radar challenge) before it will
        // issue a session. The challenge cookies were persisted server-side,
        // so route to the matching challenge/enrollment UI. A user who isn't
        // yet enrolled lands on the enrollment step and can set up MFA here —
        // otherwise they could never sign in to that org. Keep the button in
        // its loading state since we're navigating away.
        if (!result.success && "challengeType" in result) {
          const redirectSuffix = redirectParam
            ? `&redirect=${encodeURIComponent(redirectParam)}`
            : "";
          window.location.href = `${SIGN_IN_URL}?challenge=${result.challengeType}${redirectSuffix}`;
          return true;
        }

        if (!result.success) {
          setError(result.message);
          setIsLoading(false);
          toast.error(result.message);

          // If session expired, redirect to sign-in to clear stale state
          if (result.code === AuthErrorCode.PENDING_SESSION_EXPIRED) {
            router.push("/auth/sign-in" as Route);
          }

          return false;
        }

        // On success, redirect to the original deep link or dashboard
        // Rewrite the workspace slug in the redirect URL to match the selected workspace
        // Use window.location.href for a full page load to avoid stale client state
        // Fall back to sessionStorage if the URL param was lost (Safari)
        const deepLink = redirectParam || consumeRedirectUrl();
        const resolvedUrl = resolveRedirectUrl(deepLink, result.workspaceSlug);
        window.location.href = resolvedUrl || result.redirectTo;
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
    [isLoading, setError, router, redirectParam],
  );

  const handleSubmit = useCallback(async () => {
    await submit(selectedOrgId);
  }, [submit, selectedOrgId]);

  return (
    <DialogContainer
      className="border border-gray-6"
      contentClassName="bg-gray-1 border-none"
      isOpen={isOpen}
      onOpenChange={(open) => {
        if (!open && !isLoading) {
          handleClose();
        }
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
            <div className="flex flex-col gap-4 focus:outline-none!">
              <label
                htmlFor="workspace-selector"
                className="text-sm font-medium text-gray-11 focus:outline-none!"
              >
                Workspace
              </label>
              <Select
                items={sortedOrgs.map((org) => ({ value: org.id, label: org.name }))}
                value={selectedOrgId}
                onValueChange={(value) => {
                  if (value !== null) {
                    setSelectedOrgId(value);
                  }
                }}
                disabled={isLoading}
              >
                <SelectTrigger
                  id="workspace-selector"
                  className="bg-gray-2 text-gray-12 border border-gray-6 focus:outline-none! focus:ring-0 focus:border-gray-8"
                >
                  <SelectValue placeholder="Select a workspace..." className="text-gray-11" />
                </SelectTrigger>
                <SelectContent className="overflow-y-auto max-h-100 bg-gray-1 text-gray-12 focus:outline-none! border focus:border-gray-8 border-gray-6">
                  {sortedOrgs.map((org) => (
                    <SelectItem
                      key={org.id}
                      value={org.id}
                      className="text-gray-12 data-highlighted:bg-gray-3 data-highlighted:text-gray-12 focus:outline-none!"
                    >
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
              variant="primary"
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
