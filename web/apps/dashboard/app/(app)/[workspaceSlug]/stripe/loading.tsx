import { PageLoading } from "@/components/dashboard/page-loading";

/**
 * Instant feedback for the /stripe/checkout and /stripe/portal hand-off pages,
 * which block on server-side Stripe calls before redirecting. Without this the
 * navigation appears to hang on the previous page until Stripe responds.
 */
export default function Loading() {
  return <PageLoading message="Redirecting to Stripe..." />;
}
