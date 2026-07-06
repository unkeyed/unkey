"use client";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import { trpc } from "@/lib/trpc/client";
import {
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SettingCard,
  toast,
} from "@unkey/ui";
import { useMemo, useState } from "react";

const DEFAULT_CARD = "__default__";

/**
 * Attach an end-user to billing without leaving the Monetization page: search
 * an identity, fill in customer details, and we create the Stripe customer on
 * the connected account and bind it. The details are forwarded to Stripe and
 * NOT stored by Unkey — only the returned customer id is persisted.
 */
export const AttachBillingCard: React.FC<{ isAdmin: boolean }> = ({ isAdmin }) => {
  const utils = trpc.useUtils();
  const [search, setSearch] = useState("");
  const [identityId, setIdentityId] = useState("");
  const [identityLabel, setIdentityLabel] = useState("");
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [description, setDescription] = useState("");
  const [rateCard, setRateCard] = useState(DEFAULT_CARD);

  const connect = trpc.billing.stripeConnect.get.useQuery(undefined, { staleTime: 30_000 });
  const linked = connect.data?.status === "linked";

  const results = trpc.identity.search.useQuery(
    { query: search },
    { enabled: search.trim().length > 0, staleTime: 10_000 },
  );
  const cards = trpc.billing.rateCards.list.useQuery(undefined, { staleTime: 30_000 });

  const options = useMemo<ComboboxOption[]>(() => {
    const found = (results.data?.identities ?? []).map((i) => ({
      label: i.externalId,
      value: i.id,
    }));
    // Keep the chosen identity visible even after the search text changes and
    // it drops out of the latest results.
    if (identityId && !found.some((o) => o.value === identityId)) {
      return [{ label: identityLabel || identityId, value: identityId }, ...found];
    }
    return found;
  }, [results.data, identityId, identityLabel]);

  const reset = () => {
    setSearch("");
    setIdentityId("");
    setIdentityLabel("");
    setEmail("");
    setName("");
    setPhone("");
    setDescription("");
    setRateCard(DEFAULT_CARD);
  };

  const create = trpc.billing.monetization.createCustomerForIdentity.useMutation({
    onSuccess: (res) => {
      utils.identity.getBilling.invalidate({ identityId });
      toast.success(`Created Stripe customer ${res.customerId} for ${res.externalId}.`);
      reset();
    },
    onError: (err) => toast.error(err.message),
  });

  const submit = () =>
    create.mutate({
      identityId,
      email,
      name: name.trim() || undefined,
      phone: phone.trim() || undefined,
      description: description.trim() || undefined,
      rateCardId: rateCard === DEFAULT_CARD ? null : rateCard,
    });

  const disabled = !isAdmin || !linked;
  const canSubmit = !disabled && identityId !== "" && email.trim() !== "" && !create.isLoading;

  return (
    <SettingCard
      title="Attach billing to a user"
      description="Create a Stripe customer for an end-user and bill them. Customer details go to Stripe and are not stored by Unkey - only the customer ID is saved on the identity."
      border="both"
      className="items-start"
      contentWidth="w-full"
    >
      <div className="flex w-full flex-col gap-4">
        {linked ? null : (
          <div className="rounded-lg border border-warningA-4 bg-warningA-2 p-3 text-xs text-warning-11">
            Connect a Stripe account above before attaching users to billing.
          </div>
        )}

        <div className="flex flex-col gap-1">
          <span className="text-sm text-gray-12">End-user</span>
          <span className="text-xs text-gray-9">Search your identities by external ID.</span>
          <Combobox
            className="mt-1"
            options={options}
            value={identityId}
            onSelect={(v) => {
              setIdentityId(v);
              const opt = options.find((o) => o.value === v);
              setIdentityLabel(opt ? String(opt.label) : v);
            }}
            onChange={(e) => setSearch(e.currentTarget.value)}
            placeholder="Search identities..."
            emptyMessage={
              search.trim() === "" ? "Type to search identities" : "No identities found"
            }
            disabled={disabled}
          />
        </div>

        <FormInput
          type="email"
          requirement="required"
          label="Customer email"
          description="Sent to Stripe; not stored by Unkey."
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="user@example.com"
          disabled={disabled}
        />
        <FormInput
          label="Name (optional)"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Acme Inc."
          disabled={disabled}
        />
        <FormInput
          label="Phone (optional)"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          placeholder="+1 555 0100"
          disabled={disabled}
        />
        <FormInput
          label="Description (optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Notes for this customer"
          disabled={disabled}
        />

        <div className="flex flex-col gap-1">
          <span className="text-sm text-gray-12">Rate card (optional)</span>
          <span className="text-xs text-gray-9">
            Assign a card now, or leave on the workspace default.
          </span>
          <Select value={rateCard} onValueChange={setRateCard} disabled={disabled}>
            <SelectTrigger className="mt-1">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={DEFAULT_CARD}>Workspace default</SelectItem>
              {(cards.data?.rateCards ?? []).map((c) => (
                <SelectItem key={c.id} value={c.id}>
                  {c.name}
                  {c.isDefault ? " (default)" : ""}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex justify-end">
          <Button
            variant="primary"
            disabled={!canSubmit}
            loading={create.isLoading}
            onClick={submit}
          >
            Create customer & attach
          </Button>
        </div>
      </div>
    </SettingCard>
  );
};
