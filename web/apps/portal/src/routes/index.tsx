import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { z } from "zod";
import { getDefaultTabHref } from "~/lib/scopes";
import { exchangeCode, sessionQueryOptions } from "~/lib/session";

const searchSchema = z.object({
  code: z.string().optional(),
});

export const Route = createFileRoute("/")({
  validateSearch: searchSchema,
  component: PortalEntry,
});

function PortalEntry() {
  const { code } = Route.useSearch();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const exchange = useQuery({
    queryKey: ["portal", "exchange", code],
    queryFn: async () => {
      const result = await exchangeCode({ data: code ?? "" });
      if (!result.success) {
        return result;
      }
      const sessionData = await queryClient.fetchQuery({ ...sessionQueryOptions, staleTime: 0 });
      const defaultTab = sessionData ? getDefaultTabHref(sessionData.session.scopes) : null;
      await navigate({ to: defaultTab ?? "/keys", replace: true });
      return result;
    },
    enabled: Boolean(code),
    retry: false,
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: 0,
  });

  const message = code
    ? exchange.isError
      ? "Something went wrong. Please try again."
      : exchange.data && !exchange.data.success
        ? exchange.data.error
        : null
    : "No session provided. Please access this portal through your application.";

  if (message) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="max-w-md px-4 text-center">
          <h1 className="font-semibold text-2xl text-gray-12">
            {code ? "Session expired or invalid" : "Invalid access"}
          </h1>
          <p className="mt-2 text-gray-11">{message}</p>
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
