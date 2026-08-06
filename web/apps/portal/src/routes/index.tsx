import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { getDefaultTabHref } from "~/lib/permissions";
import { exchangeCode, getSessionWithPortal } from "~/lib/session";

export const Route = createFileRoute("/")({
  component: PortalEntry,
});

type ExchangeState = { status: "loading" } | { status: "error"; title: string; message: string };

function PortalEntry() {
  const navigate = useNavigate();
  const started = useRef(false);
  const [state, setState] = useState<ExchangeState>({ status: "loading" });

  useEffect(() => {
    // React may run effects twice in development. Only the first invocation can
    // read the query because it is deliberately removed immediately.
    if (started.current) {
      return;
    }
    started.current = true;

    const code = new URLSearchParams(window.location.search).get("code");
    window.history.replaceState(null, "", `${window.location.pathname}${window.location.hash}`);

    if (!code) {
      setState({
        status: "error",
        title: "Invalid access",
        message: "No session provided. Please access this portal through your application.",
      });
      return;
    }

    exchangeCode({ data: code })
      .then(async (result) => {
        if (!result.success) {
          setState({
            status: "error",
            title: "Session expired or invalid",
            message: result.error,
          });
          return;
        }
        // Session exchanged — resolve permissions to pick the correct landing tab.
        const sessionData = await getSessionWithPortal();
        const defaultTab = sessionData
          ? getDefaultTabHref(sessionData.session.permissions)
          : "/keys";
        navigate({ to: defaultTab ?? "/keys" });
      })
      .catch(() => {
        setState({
          status: "error",
          title: "Session expired or invalid",
          message: "Something went wrong. Please request a new session from your application.",
        });
      });
  }, [navigate]);

  if (state.status === "error") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="max-w-md px-4 text-center">
          <h1 className="font-semibold text-2xl text-gray-12">{state.title}</h1>
          <p className="mt-2 text-gray-11">{state.message}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <p className="text-gray-11">Authenticating...</p>
      </div>
    </div>
  );
}
