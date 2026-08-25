import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { z } from "zod";
import { getDefaultTabHref } from "~/lib/scopes";
import { exchangeCode, getSessionWithConfig } from "~/lib/session";

const searchSchema = z.object({
  code: z.string().optional(),
});

export const Route = createFileRoute("/")({
  validateSearch: searchSchema,
  component: PortalEntry,
});

type ExchangeState = { status: "loading" } | { status: "error"; message: string };

function PortalEntry() {
  const { code } = Route.useSearch();
  const navigate = useNavigate();
  const [state, setState] = useState<ExchangeState>({ status: "loading" });

  useEffect(() => {
    if (!code) {
      setState({
        status: "error",
        message: "No session provided. Please access this portal through your application.",
      });
      return;
    }

    exchangeCode({ data: code })
      .then(async (result) => {
        if (!result.success) {
          setState({ status: "error", message: result.error });
          return;
        }
        // Code exchanged — resolve scopes to pick the correct landing tab.
        const sessionData = await getSessionWithConfig();
        const defaultTab = sessionData ? getDefaultTabHref(sessionData.session.scopes) : "/keys";
        navigate({ to: defaultTab ?? "/keys" });
      })
      .catch(() => {
        setState({
          status: "error",
          message: "Something went wrong. Please try again.",
        });
      });
  }, [code, navigate]);

  if (state.status === "error") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="max-w-md px-4 text-center">
          <h1 className="font-semibold text-2xl text-gray-12">
            {code ? "Session expired or invalid" : "Invalid access"}
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
