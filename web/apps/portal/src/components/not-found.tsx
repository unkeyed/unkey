import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { PortalShell } from "~/components/portal-shell";
import { Button } from "~/components/ui/button";
import { getDefaultTabHref } from "~/lib/scopes";
import { sessionQueryOptions } from "~/lib/session";

export function NotFound() {
  const { data, isPending } = useQuery(sessionQueryOptions);

  const session = data?.session ?? null;
  const appName = data?.config?.displayName;
  const home = session ? getDefaultTabHref(session.scopes) : null;

  return (
    <PortalShell session={session} portal={data?.config ?? null} pending={isPending}>
      <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col items-center justify-center px-4 py-16 text-center sm:px-8">
        <p className="font-mono text-gray-9 text-xs tracking-wide" aria-hidden="true">
          404
        </p>
        <h1 className="mt-2 font-semibold text-gray-12 text-xl">Page not found</h1>
        <p className="mt-1 max-w-sm text-gray-11 text-sm">
          The page you asked for doesn't exist in this portal. It may have moved, or the link was
          typed wrong.
        </p>
        {!isPending && session && (
          <div className="mt-6 flex flex-wrap items-center justify-center gap-2">
            {home && <Button render={<Link to={home}>Go to API keys</Link>} />}
            {session.returnUrl && (
              <Button
                variant="outline"
                render={<a href={session.returnUrl}>Return to {appName ?? "application"}</a>}
              />
            )}
          </div>
        )}
      </main>
    </PortalShell>
  );
}
