"use client";

import { toast } from "@unkey/ui";
import type { Route } from "next";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef } from "react";

const GENERIC_ERROR = "We couldn't process your invitation. Ask for a new one.";
const MAX_RETRIES = 2;
const TOAST_DURATION_MS = 10_000;

/**
 * The shape POST /api/auth/invitation is contracted to return. `switched: false`
 * means the invitation was accepted but the org could not be made active, so the
 * user is a member without being in the workspace yet.
 */
type InvitationResponse =
  | { success: true; organizationId?: string; switched?: boolean }
  | { success: false; error?: string };

/**
 * Reads the response body without trusting it. The endpoint is our own, but a
 * proxy or an error page can return anything, and an unparsed body would let
 * arbitrary text reach a toast.
 */
function parseInvitationResponse(body: unknown): InvitationResponse {
  if (typeof body !== "object" || body === null || !("success" in body)) {
    return { success: false };
  }

  const { success } = body as { success: unknown };
  if (success !== true) {
    const error = "error" in body ? (body as { error: unknown }).error : undefined;
    return { success: false, error: typeof error === "string" ? error : undefined };
  }

  const switched = "switched" in body ? (body as { switched: unknown }).switched : undefined;
  return { success: true, switched: typeof switched === "boolean" ? switched : undefined };
}

/**
 * Accepts the invitation named by the `invitation_token` query parameter once the
 * user has a session, then removes the token from the URL.
 *
 * Renders nothing. All feedback goes through toasts, because the pages that mount
 * this have no slot for invitation state and a silent failure here is exactly the
 * bug this flow keeps regressing into: the user lands on the dashboard with no
 * workspace and no explanation.
 */
export function PostAuthInvitationHandler() {
  const router = useRouter();
  const searchParams = useSearchParams();
  // Refs, not state: this component renders null, so these only guard
  // re-entry across renders and must be readable by the retry closures. As
  // state they would be captured per render, so the in-flight check would read
  // a stale `false` and never actually block a second acceptance attempt.
  const isProcessing = useRef(false);
  const hasProcessed = useRef(false);

  useEffect(() => {
    if (hasProcessed.current) {
      return;
    }

    const invitationToken = searchParams?.get("invitation_token");

    if (!invitationToken) {
      return;
    }

    // Let the page settle before firing: the session cookie may still be
    // landing from the auth redirect that brought the user here.
    const timer = setTimeout(() => {
      processInvitation(invitationToken);
    }, 500);

    return () => clearTimeout(timer);
  }, [searchParams]);

  const processInvitation = async (invitationToken: string, retryCount = 0) => {
    // Acceptance is irreversible, so never let two attempts overlap.
    if (isProcessing.current) {
      return;
    }
    isProcessing.current = true;

    try {
      const response = await fetch("/api/auth/invitation", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ invitationToken }),
        credentials: "include", // Ensure cookies are included
      });

      const result = parseInvitationResponse(await response.json());

      if (!result.success) {
        // The session may not have propagated yet on the first attempt.
        if (response.status === 401 && retryCount < MAX_RETRIES) {
          setTimeout(() => {
            processInvitation(invitationToken, retryCount + 1);
          }, 1000);
          return;
        }

        // Only 400 carries a message the domain layer vetted for a user. The
        // other statuses answer with plumbing text ("Internal server error"),
        // which is what ENG-3014 was about, so they get the generic line.
        const message = response.status === 400 ? (result.error ?? GENERIC_ERROR) : GENERIC_ERROR;
        toast.error(message, { duration: TOAST_DURATION_MS });
        return;
      }

      // Remove invitation_token from URL
      const newSearchParams = new URLSearchParams(searchParams?.toString());
      newSearchParams.delete("invitation_token");

      const newUrl = newSearchParams.toString()
        ? `${window.location.pathname}?${newSearchParams.toString()}`
        : window.location.pathname;

      router.replace(newUrl as Route);

      // The invitation was accepted but the org could not be made active, so a
      // reload would drop the user back into their previous workspace (or into
      // workspace creation, if this was their first org) with no explanation.
      // Membership is real and the switcher now lists the org, so say so
      // instead of reloading into the wrong place.
      if (result.switched === false) {
        toast.info(
          "You've joined the workspace. Select it from the workspace switcher to open it.",
          {
            duration: TOAST_DURATION_MS,
          },
        );
        return;
      }

      // Force a page reload to ensure the new organization context is loaded
      setTimeout(() => {
        window.location.reload();
      }, 100);
    } catch (_error) {
      // Retry on network errors if we haven't retried too many times
      if (retryCount < MAX_RETRIES) {
        setTimeout(() => {
          processInvitation(invitationToken, retryCount + 1);
        }, 2000);
        return;
      }

      // This branch catches internal failures (fetch, JSON parse). Their
      // messages are not written for users and must never be shown.
      toast.error(GENERIC_ERROR, { duration: TOAST_DURATION_MS });
    } finally {
      // Released here so a scheduled retry can run, while hasProcessed stops
      // the effect from starting an independent second attempt.
      isProcessing.current = false;
      hasProcessed.current = true;
    }
  };

  // This component doesn't render anything visible
  return null;
}
