"use client";

import { exchangeSession } from "@/lib/session";
import { useEffect, useState } from "react";

type ExchangeState = { status: "loading" } | { status: "error"; message: string };

export function SessionExchange({ sessionId }: { sessionId: string }) {
  const [state, setState] = useState<ExchangeState>({ status: "loading" });

  useEffect(() => {
    exchangeSession(sessionId).then((result) => {
      if (!result.success) {
        setState({ status: "error", message: result.error });
      }
      // On success, the server action sets the cookie and triggers a redirect.
      // The component won't re-render because navigation happens server-side.
    });
  }, [sessionId]);

  if (state.status === "error") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-semibold text-gray-12">Session expired or invalid</h1>
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
