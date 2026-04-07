import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { exchangeSession, getSessionToken } from "~/lib/session";
import { getDefaultTabHref } from "~/lib/permissions";
import { useEffect, useState } from "react";
import { z } from "zod";

const searchSchema = z.object({
  session: z.string().optional(),
});

export const Route = createFileRoute("/")({
  validateSearch: searchSchema,
  component: PortalEntry,
});

type ExchangeState =
  | { status: "loading" }
  | { status: "error"; message: string };

function PortalEntry() {
  const { session: sessionId } = Route.useSearch();
  const navigate = useNavigate();
  const [state, setState] = useState<ExchangeState>({ status: "loading" });

  useEffect(() => {
    if (!sessionId) {
      setState({
        status: "error",
        message: "No session provided. Please access this portal through your application.",
      });
      return;
    }

    exchangeSession({ data: sessionId }).then((result) => {
      if (!result.success) {
        setState({ status: "error", message: result.error });
        return;
      }
      // Session exchanged — redirect to first available tab.
      // For now default to /keys since we don't have permissions from the exchange response.
      navigate({ to: "/keys" });
    });
  }, [sessionId, navigate]);

  if (state.status === "error") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center max-w-md px-4">
          <h1 className="text-2xl font-semibold text-gray-12">
            {sessionId ? "Session expired or invalid" : "Invalid access"}
          </h1>
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
