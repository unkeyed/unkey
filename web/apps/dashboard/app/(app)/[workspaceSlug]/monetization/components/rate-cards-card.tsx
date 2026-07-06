"use client";
import { trpc } from "@/lib/trpc/client";
import { Badge, Button, Empty, InfoTooltip, Loading, SettingCard, toast } from "@unkey/ui";
import { useState } from "react";
import { CreateRateCardDialog } from "./create-rate-card-dialog";

const ADMIN_ONLY_TOOLTIP = "Admin access required to manage billing";

type RateCardConfig = {
  verifications?: unknown[];
  credits?: unknown[];
  ratelimits?: unknown[];
};

// Which dimensions a card prices, for a compact summary line.
function pricedDimensions(config: unknown): string[] {
  const c = (config ?? {}) as RateCardConfig;
  const dims: string[] = [];
  if (c.verifications?.length) {
    dims.push("verifications");
  }
  if (c.credits?.length) {
    dims.push("credits");
  }
  if (c.ratelimits?.length) {
    dims.push("ratelimits");
  }
  return dims;
}

/**
 * Manage the workspace's rate cards: create tiered cards and choose the default
 * that prices every end-user (selection/assignment override it per identity).
 * This is where a customer sets HOW MUCH their users are charged.
 */
export const RateCardsCard: React.FC<{ isAdmin: boolean }> = ({ isAdmin }) => {
  const utils = trpc.useUtils();
  const [createOpen, setCreateOpen] = useState(false);

  const { data, isLoading, isError, error } = trpc.billing.rateCards.list.useQuery(undefined, {
    staleTime: 30_000,
  });

  const setDefault = trpc.billing.rateCards.setDefault.useMutation({
    onSuccess: () => {
      utils.billing.rateCards.list.invalidate();
      toast.success("Default rate card updated.");
    },
    onError: (err) => toast.error(err.message),
  });

  const body = () => {
    if (isLoading) {
      return (
        <div className="flex w-full justify-center py-8">
          <Loading />
        </div>
      );
    }
    if (isError) {
      return (
        <Empty>
          <Empty.Title>Couldn't load rate cards</Empty.Title>
          <Empty.Description>{error?.message ?? "Please try again in a moment."}</Empty.Description>
        </Empty>
      );
    }

    const cards = data?.rateCards ?? [];
    if (cards.length === 0) {
      return (
        <Empty>
          <Empty.Title>No rate cards yet</Empty.Title>
          <Empty.Description>
            Create a rate card to set how much your end-users are charged, then make one the default
            so it applies to everyone.
          </Empty.Description>
        </Empty>
      );
    }

    return (
      <div className="flex w-full flex-col gap-3">
        {cards.map((card) => (
          <div
            key={card.id}
            className="flex items-center justify-between gap-4 rounded-lg border border-grayA-4 p-3"
          >
            <div className="flex min-w-0 flex-col gap-1">
              <div className="flex items-center gap-2">
                <span className="truncate text-sm text-gray-12">{card.name}</span>
                <span className="text-xs text-gray-9">{card.currency}</span>
                {card.isDefault ? <Badge variant="primary">Default</Badge> : null}
                {card.selectable ? <Badge variant="secondary">Selectable</Badge> : null}
              </div>
              <span className="text-xs text-gray-9">
                Prices: {pricedDimensions(card.config).join(", ") || "none"}
              </span>
            </div>
            {card.isDefault ? (
              <span className="text-xs text-gray-9">Applies to all end-users</span>
            ) : (
              <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
                <span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={!isAdmin || setDefault.isLoading}
                    onClick={() => setDefault.mutate({ rateCardId: card.id })}
                  >
                    Set as default
                  </Button>
                </span>
              </InfoTooltip>
            )}
          </div>
        ))}
      </div>
    );
  };

  return (
    <>
      <SettingCard
        title="Rate cards"
        description="Tiered prices for your end-users' usage. The default card prices everyone; assign a different card per identity to override it."
        border="both"
        className="items-start"
        contentWidth="w-full"
      >
        <div className="flex w-full flex-col gap-4">
          {body()}
          <div className="flex justify-end">
            <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
              <span>
                <Button variant="primary" disabled={!isAdmin} onClick={() => setCreateOpen(true)}>
                  Create rate card
                </Button>
              </span>
            </InfoTooltip>
          </div>
        </div>
      </SettingCard>
      <CreateRateCardDialog isOpen={createOpen} onOpenChange={setCreateOpen} />
    </>
  );
};
