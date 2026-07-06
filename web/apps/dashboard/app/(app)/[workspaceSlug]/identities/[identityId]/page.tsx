import { IdentityBillingCard } from "./components/identity-billing-card";
import { IdentityDetailsLogsClient } from "./logs-client";
import { Navigation } from "./navigation";

export default async function IdentityDetailsPage(props: {
  params: Promise<{ identityId: string }>;
}) {
  const { identityId } = await props.params;

  return (
    <div className="w-full">
      <Navigation identityId={identityId} />
      <IdentityBillingCard identityId={identityId} />
      <IdentityDetailsLogsClient identityId={identityId} />
    </div>
  );
}
