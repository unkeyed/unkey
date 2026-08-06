import { createFileRoute, redirect } from "@tanstack/react-router";
import type { SortingState } from "@tanstack/react-table";
import { AlertTriangle } from "lucide-react";
import { useState } from "react";
import { useKeysListQuery } from "~/components/keys-table/hooks/queries/use-keys-list-query";
import { useRerollKey } from "~/components/keys-table/hooks/use-reroll-key";
import { KeysTable } from "~/components/keys-table/keys-table";
import type { StatusFilter } from "~/components/keys-table/keys-toolbar";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import { Button } from "~/components/ui/button";
import { TooltipProvider } from "~/components/ui/tooltip";
import { canReadKeys } from "~/lib/permissions";
import { isUnauthorizedError } from "~/lib/portal-api";

export const Route = createFileRoute("/_portal/keys")({
  beforeLoad: ({ context }) => {
    // The page lists keys via portal.listKeys (authorized with read_key), so it
    // must only render for sessions that carry that action.
    if (!canReadKeys(context.session.permissions)) {
      throw redirect({ to: "/" });
    }
  },
  component: KeysPage,
});

function KeysPage() {
  const { portal } = Route.useRouteContext();
  const { keys, isInitialLoading, isError, error, refetch } = useKeysListQuery();
  const reroll = useRerollKey();

  // Search, filter, sort, and pagination are handled client-side by the table
  // over the full key set (portal end users have few keys).
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [sorting, setSorting] = useState<SortingState>([]);
  const [page, setPage] = useState(0);

  return (
    <main className="mx-auto max-w-5xl px-4 pt-8 pb-12 sm:px-8">
      {isInitialLoading ? (
        <KeysLoading />
      ) : isError && isUnauthorizedError(error) ? (
        // Expired/invalid session: retrying won't help — point the user back to
        // the application that launched the portal.
        <SessionExpired returnUrl={portal?.returnUrl ?? null} />
      ) : isError ? (
        <KeysError message={error instanceof Error ? error.message : undefined} onRetry={refetch} />
      ) : (
        <TooltipProvider delay={300}>
          <KeysTable
            appName={portal?.slug ?? undefined}
            keys={keys}
            searchValue={search}
            onSearchChange={setSearch}
            statusValue={status}
            onStatusChange={setStatus}
            sorting={sorting}
            onSortingChange={setSorting}
            pageIndex={page}
            onPageChange={setPage}
            onReroll={(input) => reroll.mutateAsync(input)}
          />
        </TooltipProvider>
      )}
    </main>
  );
}

function KeysLoading() {
  return (
    <div
      className="flex min-h-64 items-center justify-center text-gray-11 text-sm"
      aria-busy="true"
    >
      Loading keys…
    </div>
  );
}

function SessionExpired({ returnUrl }: { returnUrl: string | null }) {
  return (
    <div className="flex flex-col items-center gap-4">
      <Alert className="max-w-md">
        <AlertTriangle />
        <AlertTitle>Your session has expired</AlertTitle>
        <AlertDescription>
          Return to your application to continue managing your API keys.
        </AlertDescription>
      </Alert>
      {returnUrl && (
        <Button variant="outline" render={<a href={returnUrl}>Back to application</a>} />
      )}
    </div>
  );
}

function KeysError({ message, onRetry }: { message?: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center gap-4">
      <Alert variant="destructive" className="max-w-md">
        <AlertTriangle />
        <AlertTitle>Couldn't load your keys</AlertTitle>
        <AlertDescription>{message ?? "Something went wrong. Please try again."}</AlertDescription>
      </Alert>
      <Button variant="outline" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}
