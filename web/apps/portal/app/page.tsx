import { resolvePortalConfig } from "@/lib/portal-config";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import { SessionExchange } from "./session-exchange";

/**
 * Portal entry point.
 * Reads the `?session` query param and exchanges it for a browser session.
 * If no session param is present, shows an error page.
 */
export default async function PortalEntryPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const sessionId = typeof params.session === "string" ? params.session : null;

  const headersList = await headers();
  const hostname = headersList.get("host")?.split(":")[0] ?? "";
  const portalConfig = await resolvePortalConfig(hostname);

  if (!portalConfig) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-semibold text-gray-12">Portal not found</h1>
          <p className="mt-2 text-gray-11">This portal does not exist or has not been configured.</p>
        </div>
      </div>
    );
  }

  if (!portalConfig.enabled) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-semibold text-gray-12">Portal unavailable</h1>
          <p className="mt-2 text-gray-11">This portal is currently unavailable. Please contact support.</p>
        </div>
      </div>
    );
  }

  if (!sessionId) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-semibold text-gray-12">Invalid access</h1>
          <p className="mt-2 text-gray-11">
            No session provided. Please access this portal through your application.
          </p>
        </div>
      </div>
    );
  }

  return <SessionExchange sessionId={sessionId} />;
}
