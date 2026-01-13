import { Navigation } from "./navigation";
import { IdentityDetailsLogsClient } from "./logs-client";

export default function IdentityDetailsPage(props: {
  params: { identityId: string };
}) {
  const { identityId } = props.params;

  return (
    <div className="w-full">
      <Navigation identityId={identityId} />
      <IdentityDetailsLogsClient identityId={identityId} />
    </div>
  );
}
