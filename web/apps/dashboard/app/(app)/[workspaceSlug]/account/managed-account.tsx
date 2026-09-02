"use client";

import "@unkey/workos-widgets/styles.css";
import { logManagedAuthOutcome } from "@/lib/auth/telemetry";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { ManagedUserWidgets } from "@unkey/workos-widgets";
import { useAccessToken, useAuth } from "@workos-inc/authkit-nextjs/components";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { AccountSettingsSkeleton } from "./account-settings-skeleton";
import { AccountShell } from "./account-shell";
import { AccountUnavailable } from "./account-unavailable";

type RefreshIdentity = () => Promise<boolean>;

export function ManagedAccount() {
  const { user, impersonator, loading, getAuth, refreshAuth } = useAuth();
  const utils = trpc.useUtils();
  const refreshInFlightRef = useRef<Promise<boolean> | null>(null);

  const refreshIdentity = useCallback<RefreshIdentity>(() => {
    if (refreshInFlightRef.current) {
      return refreshInFlightRef.current;
    }

    const refreshPromise = (async () => {
      const result = await refreshAuth();
      if (result && "error" in result) {
        logManagedAuthOutcome("session_refresh", "failure");
        return false;
      }
      await utils.user.getCurrentUser.invalidate();
      logManagedAuthOutcome("session_refresh", "success");
      return true;
    })()
      .catch(() => {
        logManagedAuthOutcome("session_refresh", "failure");
        return false;
      })
      .finally(() => {
        refreshInFlightRef.current = null;
      });

    refreshInFlightRef.current = refreshPromise;
    return refreshPromise;
  }, [refreshAuth, utils]);

  const syncIdentity = useCallback(async () => {
    await getAuth();
    await utils.user.getCurrentUser.invalidate();
  }, [getAuth, utils]);

  let content: React.ReactNode;
  if (loading) {
    content = <AccountSettingsSkeleton />;
  } else if (!user) {
    content = (
      <AccountError
        heading="Your session has expired"
        description="Sign in again to manage your account."
        action={
          <Link className="underline" href={routes.auth.signIn()}>
            Sign in again
          </Link>
        }
      />
    );
  } else if (impersonator) {
    content = <AccountUnavailable reason="impersonation" />;
  } else {
    content = (
      <ManagedAccountWidgets refreshIdentity={refreshIdentity} syncIdentity={syncIdentity} />
    );
  }

  return <AccountShell>{content}</AccountShell>;
}

function ManagedAccountWidgets({
  refreshIdentity,
  syncIdentity,
}: {
  refreshIdentity: RefreshIdentity;
  syncIdentity: () => Promise<void>;
}) {
  const { accessToken, loading, error, refresh, getAccessToken } = useAccessToken();
  const [retryError, setRetryError] = useState<string | null>(null);
  const [retrying, setRetrying] = useState(false);

  const getWidgetAccessToken = useCallback(async () => {
    try {
      const token = await getAccessToken();
      if (!token) {
        throw new Error("Session expired");
      }
      logManagedAuthOutcome("widget_token", "success");
      return token;
    } catch {
      logManagedAuthOutcome("widget_token", "failure");
      throw new Error("Your session has expired.");
    }
  }, [getAccessToken]);

  useEffect(() => {
    return () => {
      void refreshIdentity();
    };
  }, [refreshIdentity]);

  useEffect(() => {
    if (error) {
      logManagedAuthOutcome("widget_token", "failure");
    }
  }, [error]);

  if (loading || (!accessToken && !error && !retryError)) {
    return <AccountSettingsSkeleton />;
  }

  if (error || retryError) {
    return (
      <AccountError
        heading="Account settings are temporarily unavailable"
        description={retryError ?? "WorkOS could not load your account settings."}
        action={
          <button
            type="button"
            disabled={retrying}
            className="underline disabled:opacity-50"
            onClick={async () => {
              setRetrying(true);
              setRetryError(null);
              try {
                const token = await refresh();
                if (!token) {
                  throw new Error("Session expired");
                }
                logManagedAuthOutcome("widget_token", "success");
                await syncIdentity();
                requestAnimationFrame(() =>
                  document.getElementById("profile-settings-heading")?.focus(),
                );
              } catch {
                logManagedAuthOutcome("widget_token", "failure");
                setRetryError("We could not refresh your account session. Sign in again.");
              } finally {
                setRetrying(false);
              }
            }}
          >
            {retrying ? "Retrying..." : "Retry"}
          </button>
        }
      />
    );
  }

  return <ManagedUserWidgets getAccessToken={getWidgetAccessToken} />;
}

function AccountError({
  heading,
  description,
  action,
}: {
  heading: string;
  description: string;
  action: React.ReactNode;
}) {
  const headingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    headingRef.current?.focus();
  }, []);

  return (
    <section className="rounded-lg border border-error-6 bg-error-2 p-6" role="alert">
      <h2 ref={headingRef} tabIndex={-1} className="font-medium outline-none">
        {heading}
      </h2>
      <p className="mt-2 text-sm text-gray-11">{description}</p>
      <div className="mt-4 text-sm font-medium">{action}</div>
    </section>
  );
}
