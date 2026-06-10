"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef } from "react";

interface PostAuthInvitationHandlerProps {
  /**
   * Whether to automatically process invitations on mount
   * @default true
   */
  autoProcess?: boolean;
  /**
   * Callback fired when invitation processing completes
   */
  onComplete?: (success: boolean, error?: string) => void;
}

/**
 * Component that handles invitation processing after successful authentication.
 *
 * This component should be rendered on pages where users land after authentication
 * to automatically process any invitation tokens that were part of the auth flow.
 */
export function PostAuthInvitationHandler({
  autoProcess = true,
  onComplete,
}: PostAuthInvitationHandlerProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const isProcessingRef = useRef(false);
  const hasProcessedRef = useRef(false);

  useEffect(() => {
    if (!autoProcess || hasProcessedRef.current) {
      return;
    }

    const invitationToken = searchParams?.get("invitation_token");

    if (!invitationToken) {
      return;
    }

    // Add a small delay to ensure the page has fully loaded and session is stable
    const timer = setTimeout(() => {
      processInvitation(invitationToken);
    }, 500);

    return () => clearTimeout(timer);
  }, [autoProcess, searchParams]);

  const processInvitation = async (invitationToken: string, retryCount = 0) => {
    if (isProcessingRef.current) {
      return;
    }
    isProcessingRef.current = true;

    try {
      const response = await fetch("/api/auth/invitation", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ invitationToken }),
        credentials: "include", // Ensure cookies are included
      });

      const result = await response.json();

      if (result.success) {
        // Remove invitation_token from URL
        const newSearchParams = new URLSearchParams(searchParams?.toString());
        newSearchParams.delete("invitation_token");

        const newUrl = newSearchParams.toString()
          ? `${window.location.pathname}?${newSearchParams.toString()}`
          : window.location.pathname;

        router.replace(newUrl);

        // Force a page reload to ensure the new organization context is loaded
        setTimeout(() => {
          window.location.reload();
        }, 100);

        onComplete?.(true);
      } else {
        // If authentication failed and we haven't retried too many times, try again
        if (response.status === 401 && retryCount < 2) {
          isProcessingRef.current = false;
          setTimeout(() => {
            processInvitation(invitationToken, retryCount + 1);
          }, 1000);
          return;
        }

        onComplete?.(false, result.error);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "Unknown error";

      // Retry on network errors if we haven't retried too many times
      if (retryCount < 2) {
        isProcessingRef.current = false;
        setTimeout(() => {
          processInvitation(invitationToken, retryCount + 1);
        }, 2000);
        return;
      }

      onComplete?.(false, errorMessage);
    } finally {
      isProcessingRef.current = false;
      hasProcessedRef.current = true;
    }
  };

  // This component doesn't render anything visible
  return null;
}
