"use client";
import { trpc } from "@/lib/trpc/client";
import { Button, Checkbox, DialogContainer, FormInput, Input, toast } from "@unkey/ui";
import { useState } from "react";

type DimensionKey = "verifications" | "credits" | "ratelimits";

const DIMENSIONS: { key: DimensionKey; label: string; help: string }[] = [
  { key: "verifications", label: "Key verifications", help: "Per valid key verification." },
  { key: "credits", label: "Credits spent", help: "Per credit consumed." },
  { key: "ratelimits", label: "Ratelimit checks", help: "Per passed ratelimit check." },
];

// One editable tier row. Kept as strings while editing; parsed on submit.
// lastUnit "" = unbounded (the final tier); centsPerUnit "" = free.
type TierInput = { firstUnit: string; lastUnit: string; centsPerUnit: string };

type DimensionState = { enabled: boolean; tiers: TierInput[] };

const emptyDimension = (): DimensionState => ({
  enabled: false,
  tiers: [{ firstUnit: "1", lastUnit: "", centsPerUnit: "" }],
});

type Props = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
};

export const CreateRateCardDialog: React.FC<Props> = ({ isOpen, onOpenChange }) => {
  const utils = trpc.useUtils();
  const [name, setName] = useState("");
  const [currency, setCurrency] = useState("USD");
  const [selectable, setSelectable] = useState(false);
  const [dimensions, setDimensions] = useState<Record<DimensionKey, DimensionState>>({
    verifications: emptyDimension(),
    credits: emptyDimension(),
    ratelimits: emptyDimension(),
  });

  const reset = () => {
    setName("");
    setCurrency("USD");
    setSelectable(false);
    setDimensions({
      verifications: emptyDimension(),
      credits: emptyDimension(),
      ratelimits: emptyDimension(),
    });
  };

  const create = trpc.billing.rateCards.create.useMutation({
    onSuccess: () => {
      utils.billing.rateCards.list.invalidate();
      toast.success("Rate card created.");
      reset();
      onOpenChange(false);
    },
    onError: (err) => toast.error(err.message),
  });

  const setDim = (key: DimensionKey, next: Partial<DimensionState>) =>
    setDimensions((prev) => ({ ...prev, [key]: { ...prev[key], ...next } }));

  const updateTier = (key: DimensionKey, idx: number, patch: Partial<TierInput>) =>
    setDim(key, {
      tiers: dimensions[key].tiers.map((t, i) => (i === idx ? { ...t, ...patch } : t)),
    });

  const addTier = (key: DimensionKey) => {
    const tiers = dimensions[key].tiers;
    const prevLast = tiers[tiers.length - 1]?.lastUnit;
    const nextFirst = prevLast && prevLast.trim() !== "" ? String(Number(prevLast) + 1) : "";
    setDim(key, { tiers: [...tiers, { firstUnit: nextFirst, lastUnit: "", centsPerUnit: "" }] });
  };

  const removeTier = (key: DimensionKey, idx: number) =>
    setDim(key, { tiers: dimensions[key].tiers.filter((_, i) => i !== idx) });

  const submit = () => {
    // Build the config from enabled dimensions only. The server validates tier
    // contiguity and the single-unbounded-tail rule, so surface its error
    // rather than duplicating that math here.
    const config: Record<
      string,
      { firstUnit: number; lastUnit: number | null; centsPerUnit: string | null }[]
    > = {};
    for (const { key } of DIMENSIONS) {
      const dim = dimensions[key];
      if (!dim.enabled) {
        continue;
      }
      config[key] = dim.tiers.map((t) => ({
        firstUnit: Number(t.firstUnit),
        lastUnit: t.lastUnit.trim() === "" ? null : Number(t.lastUnit),
        centsPerUnit: t.centsPerUnit.trim() === "" ? null : t.centsPerUnit.trim(),
      }));
    }

    if (Object.keys(config).length === 0) {
      toast.error("Enable and price at least one dimension.");
      return;
    }
    create.mutate({ name, currency, config, selectable });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={(open) => {
        if (!open && create.isLoading) {
          return;
        }
        onOpenChange(open);
      }}
      title="Create rate card"
      footer={
        <Button
          variant="primary"
          size="xlg"
          className="w-full rounded-lg"
          disabled={create.isLoading || name.trim() === ""}
          loading={create.isLoading}
          onClick={submit}
        >
          Create rate card
        </Button>
      }
    >
      <div className="flex flex-col gap-5">
        <FormInput
          requirement="required"
          label="Name"
          description="How this rate card appears when assigning it to end-users."
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Standard usage"
        />
        <FormInput
          requirement="required"
          label="Currency"
          description="3-letter ISO code. Must match your connected Stripe account."
          value={currency}
          onChange={(e) => setCurrency(e.target.value.toUpperCase())}
          placeholder="USD"
        />

        {DIMENSIONS.map(({ key, label, help }) => {
          const dim = dimensions[key];
          return (
            <div key={key} className="rounded-lg border border-grayA-4 p-3">
              <div className="flex items-center gap-2">
                <Checkbox
                  checked={dim.enabled}
                  onCheckedChange={(c) => setDim(key, { enabled: Boolean(c) })}
                  aria-label={`Bill ${label}`}
                />
                <span className="text-sm text-gray-12">{label}</span>
                <span className="text-xs text-gray-9">{help}</span>
              </div>

              {dim.enabled ? (
                <div className="mt-3 flex flex-col gap-2">
                  <div className="flex gap-2 text-xs text-gray-9">
                    <span className="flex-1">From unit</span>
                    <span className="flex-1">To unit (blank = unlimited)</span>
                    <span className="flex-1">Cents / unit (blank = free)</span>
                    <span className="w-8" />
                  </div>
                  {dim.tiers.map((tier, idx) => (
                    // biome-ignore lint/suspicious/noArrayIndexKey: rows are positional and reorder only by add/remove
                    <div key={idx} className="flex items-center gap-2">
                      <Input
                        type="number"
                        min={1}
                        className="flex-1"
                        value={tier.firstUnit}
                        onChange={(e) => updateTier(key, idx, { firstUnit: e.target.value })}
                        placeholder="1"
                      />
                      <Input
                        type="number"
                        min={1}
                        className="flex-1"
                        value={tier.lastUnit}
                        onChange={(e) => updateTier(key, idx, { lastUnit: e.target.value })}
                        placeholder="unlimited"
                      />
                      <Input
                        className="flex-1"
                        value={tier.centsPerUnit}
                        onChange={(e) => updateTier(key, idx, { centsPerUnit: e.target.value })}
                        placeholder="0"
                      />
                      <Button
                        variant="outline"
                        size="sm"
                        className="w-8"
                        disabled={dim.tiers.length === 1}
                        onClick={() => removeTier(key, idx)}
                        aria-label="Remove tier"
                      >
                        -
                      </Button>
                    </div>
                  ))}
                  <Button variant="outline" size="sm" onClick={() => addTier(key)}>
                    Add tier
                  </Button>
                </div>
              ) : null}
            </div>
          );
        })}

        <div className="flex items-center gap-2">
          <Checkbox
            checked={selectable}
            onCheckedChange={(c) => setSelectable(Boolean(c))}
            aria-label="Let end-users select this card"
          />
          <span className="text-sm text-gray-12">Let end-users select this card</span>
        </div>
      </div>
    </DialogContainer>
  );
};
