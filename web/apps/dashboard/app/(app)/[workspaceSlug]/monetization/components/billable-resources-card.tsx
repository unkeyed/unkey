"use client";
import { Switch } from "@/components/ui/switch";
import { trpc } from "@/lib/trpc/client";
import { Empty, InfoTooltip, Loading, SettingCard, SettingCardGroup, toast } from "@unkey/ui";

const ADMIN_ONLY_TOOLTIP = "Admin access required to manage billing";

type ResourceType = "keyspace" | "namespace";

/**
 * Lets the customer choose which keyspaces (APIs) and ratelimit namespaces are
 * metered for end-user billing. Enablement is opt-in: only enabled resources
 * contribute to an identity's billable usage. Toggling is admin-only and
 * writes through billing.monetization.setBillableResource.
 */
export const BillableResourcesCard: React.FC<{ isAdmin: boolean }> = ({ isAdmin }) => {
  const utils = trpc.useUtils();
  const { data, isLoading, isError, error } = trpc.billing.monetization.listBillableResources.useQuery(
    undefined,
    { staleTime: 30_000 },
  );

  const setResource = trpc.billing.monetization.setBillableResource.useMutation({
    onSuccess: () => {
      utils.billing.monetization.listBillableResources.invalidate();
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });

  const toggle = (resourceType: ResourceType, resourceId: string, enabled: boolean) => {
    setResource.mutate({ resourceType, resourceId, enabled });
  };

  if (isLoading) {
    return (
      <div className="flex w-full justify-center py-8">
        <Loading />
      </div>
    );
  }

  // Surface load failures instead of silently rendering an empty list — an
  // errored query must not read as "you have no keyspaces."
  if (isError) {
    return (
      <SettingCardGroup>
        <SettingCard title="Billable resources" description="" border="both">
          <Empty>
            <Empty.Title>Couldn't load billable resources</Empty.Title>
            <Empty.Description>
              {error?.message ?? "Please try again in a moment."}
            </Empty.Description>
          </Empty>
        </SettingCard>
      </SettingCardGroup>
    );
  }

  const keyspaces = data?.keyspaces ?? [];
  const namespaces = data?.namespaces ?? [];

  return (
    <SettingCardGroup>
      <SettingCard
        title="Billable keyspaces (APIs)"
        description="Key verifications and spent credits are billed to your end-users only for the keyspaces you enable here."
        border="top"
        className="items-start"
      >
        <div className="flex w-full flex-col gap-3">
          {keyspaces.length === 0 ? (
            <Empty>
              <Empty.Description>No keyspaces yet.</Empty.Description>
            </Empty>
          ) : (
            keyspaces.map((ks) => (
              <div key={ks.resourceId} className="flex items-center justify-between gap-4">
                <span className="truncate text-sm text-gray-12">{ks.name}</span>
                <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
                  <span>
                    <Switch
                      checked={ks.enabled}
                      disabled={!isAdmin || setResource.isLoading}
                      onCheckedChange={(checked) => toggle("keyspace", ks.resourceId, checked)}
                      aria-label={`Bill usage for ${ks.name}`}
                    />
                  </span>
                </InfoTooltip>
              </div>
            ))
          )}
        </div>
      </SettingCard>

      <SettingCard
        title="Billable ratelimit namespaces"
        description="Passed ratelimit checks are billed to your end-users only for the namespaces you enable here."
        border="bottom"
        className="items-start"
      >
        <div className="flex w-full flex-col gap-3">
          {namespaces.length === 0 ? (
            <Empty>
              <Empty.Description>No ratelimit namespaces yet.</Empty.Description>
            </Empty>
          ) : (
            namespaces.map((ns) => (
              <div key={ns.resourceId} className="flex items-center justify-between gap-4">
                <span className="truncate text-sm text-gray-12">{ns.name}</span>
                <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
                  <span>
                    <Switch
                      checked={ns.enabled}
                      disabled={!isAdmin || setResource.isLoading}
                      onCheckedChange={(checked) => toggle("namespace", ns.resourceId, checked)}
                      aria-label={`Bill usage for ${ns.name}`}
                    />
                  </span>
                </InfoTooltip>
              </div>
            ))
          )}
        </div>
      </SettingCard>
    </SettingCardGroup>
  );
};
