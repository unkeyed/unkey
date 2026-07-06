"use client";
import { trpc } from "@/lib/trpc/client";
import {
  Button,
  Empty,
  Input,
  Loading,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SettingCard,
  toast,
} from "@unkey/ui";
import { useState } from "react";

const ADMIN_ONLY = "Admin access required to manage billing";
// Radix Select forbids an empty-string value, so the "use workspace default"
// choice needs a sentinel that maps back to null on save.
const DEFAULT_CARD = "__default__";

type Provider = "none" | "stripe_connect" | "export";
type RateCard = { id: string; name: string; isDefault: boolean };

type FormProps = {
  identityId: string;
  isAdmin: boolean;
  cards: RateCard[];
  initial: {
    billingProvider: Provider;
    billingExternalCustomerId: string | null;
    rateCardId: string | null;
    selectedRateCardId: string | null;
  };
};

const BillingForm: React.FC<FormProps> = ({ identityId, isAdmin, cards, initial }) => {
  const utils = trpc.useUtils();
  const [provider, setProvider] = useState<Provider>(initial.billingProvider);
  const [customerId, setCustomerId] = useState(initial.billingExternalCustomerId ?? "");
  const [rateCard, setRateCard] = useState<string>(initial.rateCardId ?? DEFAULT_CARD);

  const update = trpc.identity.update.billing.useMutation({
    onSuccess: () => {
      utils.identity.getBilling.invalidate({ identityId });
      toast.success("Billing updated for this identity.");
    },
    onError: (err) => toast.error(err.message),
  });

  const defaultCard = cards.find((c) => c.isDefault);
  const selectedName = initial.selectedRateCardId
    ? (cards.find((c) => c.id === initial.selectedRateCardId)?.name ?? initial.selectedRateCardId)
    : null;

  const save = () =>
    update.mutate({
      identityId,
      billingProvider: provider,
      billingExternalCustomerId: customerId.trim() === "" ? null : customerId.trim(),
      rateCardId: rateCard === DEFAULT_CARD ? null : rateCard,
    });

  return (
    <div className="flex w-full flex-col gap-4">
      <div className="flex flex-col gap-1">
        <span className="text-sm text-gray-12">Billing provider</span>
        <span className="text-xs text-gray-9">
          Bill this user through your connected Stripe account. "None" excludes it from billing.
        </span>
        <Select
          value={provider}
          onValueChange={(v) => setProvider(v as Provider)}
          disabled={!isAdmin}
        >
          <SelectTrigger className="mt-1">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="none">None (not billed)</SelectItem>
            <SelectItem value="stripe_connect">Stripe Connect</SelectItem>
            <SelectItem value="export">Export only</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="flex flex-col gap-1">
        <span className="text-sm text-gray-12">Stripe customer ID</span>
        <span className="text-xs text-gray-9">
          The customer on your connected account this user's invoice items post to (e.g. cus_...).
        </span>
        <Input
          className="mt-1"
          value={customerId}
          onChange={(e) => setCustomerId(e.target.value)}
          placeholder="cus_..."
          disabled={!isAdmin}
        />
      </div>

      <div className="flex flex-col gap-1">
        <span className="text-sm text-gray-12">Rate card</span>
        <span className="text-xs text-gray-9">
          Price this user with a specific card, or fall back to the workspace default
          {defaultCard ? ` (${defaultCard.name})` : " (none set yet)"}.
        </span>
        <Select value={rateCard} onValueChange={setRateCard} disabled={!isAdmin}>
          <SelectTrigger className="mt-1">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={DEFAULT_CARD}>Workspace default</SelectItem>
            {cards.map((c) => (
              <SelectItem key={c.id} value={c.id}>
                {c.name}
                {c.isDefault ? " (default)" : ""}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {selectedName ? (
          <span className="text-xs text-warning-11">
            This user self-selected "{selectedName}", which overrides the assignment above while it
            stays selectable.
          </span>
        ) : null}
      </div>

      <div className="flex justify-end">
        <Button
          variant="primary"
          disabled={!isAdmin || update.isLoading}
          loading={update.isLoading}
          onClick={save}
          title={isAdmin ? undefined : ADMIN_ONLY}
        >
          Save billing
        </Button>
      </div>
    </div>
  );
};

/**
 * Per-identity billing configuration: bind the end-user to a provider, map it
 * to a provider customer, and assign its rate card. This is the "attach billing
 * to a user" surface the Monetization page points to.
 */
export const IdentityBillingCard: React.FC<{ identityId: string }> = ({ identityId }) => {
  const billing = trpc.identity.getBilling.useQuery({ identityId }, { staleTime: 30_000 });
  const cardsQuery = trpc.billing.rateCards.list.useQuery(undefined, { staleTime: 30_000 });
  const { data: currentUser } = trpc.user.getCurrentUser.useQuery();
  const isAdmin = currentUser?.role === "admin";

  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-4">
      <SettingCard
        title="Billing"
        description="How this end-user is billed: provider binding, Stripe customer, and rate card."
        border="both"
        className="items-start"
        contentWidth="w-full"
      >
        {billing.isLoading || cardsQuery.isLoading ? (
          <div className="flex w-full justify-center py-8">
            <Loading />
          </div>
        ) : billing.isError || !billing.data ? (
          <Empty>
            <Empty.Title>Couldn't load billing</Empty.Title>
            <Empty.Description>
              {billing.error?.message ?? "Please try again in a moment."}
            </Empty.Description>
          </Empty>
        ) : (
          <BillingForm
            identityId={identityId}
            isAdmin={isAdmin}
            cards={(cardsQuery.data?.rateCards ?? []).map((c) => ({
              id: c.id,
              name: c.name,
              isDefault: c.isDefault,
            }))}
            initial={{
              billingProvider: billing.data.billingProvider,
              billingExternalCustomerId: billing.data.billingExternalCustomerId,
              rateCardId: billing.data.rateCardId,
              selectedRateCardId: billing.data.selectedRateCardId,
            }}
          />
        )}
      </SettingCard>
    </div>
  );
};
