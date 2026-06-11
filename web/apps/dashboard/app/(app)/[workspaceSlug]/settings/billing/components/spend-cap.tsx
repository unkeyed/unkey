"use client";

import { formatDollars } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, FormInput, InfoTooltip, toast } from "@unkey/ui";
import { useState } from "react";

const ADMIN_ONLY_TOOLTIP = "Admin access required to manage billing";

/**
 * Parses a whole-dollar form value into cents. Empty = no cap (null);
 * anything non-numeric or non-positive is invalid (undefined).
 */
function parseDollars(value: string): number | null | undefined {
  const trimmed = value.trim();
  if (trimmed === "") {
    return null;
  }
  if (!/^\d+$/.test(trimmed)) {
    return undefined;
  }
  const dollars = Number.parseInt(trimmed, 10);
  return dollars > 0 ? dollars * 100 : undefined;
}

function capLabel(cents: number | null): string {
  return cents === null ? "None" : `${formatDollars(cents)}/mo`;
}

type SpendCapProps = {
  isAdmin: boolean;
};

/**
 * The Compute spend-cap row: shows the stored soft (notify) and hard (stop
 * workloads) caps and opens the edit dialog. v1 stores the values only;
 * enforcement and notifications come later, so the copy is explicit that
 * nothing acts on them yet.
 */
export const SpendCap: React.FC<SpendCapProps> = ({ isAdmin }) => {
  const trpcUtils = trpc.useUtils();
  const [isOpen, setOpen] = useState(false);
  const [softInput, setSoftInput] = useState("");
  const [hardInput, setHardInput] = useState("");

  const { data: caps } = trpc.billing.getDeploySpendCaps.useQuery(undefined, {
    staleTime: 30_000,
  });

  const save = trpc.billing.setDeploySpendCaps.useMutation({
    onSuccess: async () => {
      setOpen(false);
      toast.success("Spend caps saved");
      await trpcUtils.billing.getDeploySpendCaps.invalidate();
    },
    onError: (err) => toast.error(err.message),
  });

  const softCapCents = parseDollars(softInput);
  const hardCapCents = parseDollars(hardInput);
  const invalid =
    softCapCents === undefined ||
    hardCapCents === undefined ||
    (softCapCents !== null && hardCapCents !== null && softCapCents > hardCapCents);

  const openDialog = () => {
    setSoftInput(caps?.softCapCents != null ? String(caps.softCapCents / 100) : "");
    setHardInput(caps?.hardCapCents != null ? String(caps.hardCapCents / 100) : "");
    setOpen(true);
  };

  return (
    <>
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-[13px] text-gray-12">Spend caps</p>
          <p className="truncate text-[12px] text-gray-10">
            {caps
              ? `Soft ${capLabel(caps.softCapCents)} · Hard ${capLabel(caps.hardCapCents)}`
              : "—"}
          </p>
        </div>
        <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
          <span>
            <Button variant="outline" size="sm" disabled={!isAdmin || !caps} onClick={openDialog}>
              Edit
            </Button>
          </span>
        </InfoTooltip>
      </div>

      <DialogContainer
        isOpen={isOpen}
        onOpenChange={setOpen}
        title="Compute spend caps"
        subTitle="Caps apply to usage spend per calendar month. Leave a field empty for no cap."
        footer={
          <Button
            type="button"
            variant="primary"
            size="xlg"
            className="w-full rounded-lg"
            disabled={invalid}
            loading={save.isLoading}
            onClick={() => {
              if (softCapCents === undefined || hardCapCents === undefined) {
                return;
              }
              save.mutate({ softCapCents, hardCapCents });
            }}
          >
            Save caps
          </Button>
        }
      >
        <div className="flex flex-col gap-4">
          <FormInput
            label="Soft cap"
            description="We notify you when usage spend crosses this amount."
            placeholder="No cap"
            prefix="$"
            inputMode="numeric"
            value={softInput}
            onChange={(e) => setSoftInput(e.currentTarget.value)}
            error={
              softCapCents === undefined
                ? "Enter a whole dollar amount, or leave empty."
                : undefined
            }
          />
          <FormInput
            label="Hard cap"
            description="We stop your workloads when usage spend reaches this amount."
            placeholder="No cap"
            prefix="$"
            inputMode="numeric"
            value={hardInput}
            onChange={(e) => setHardInput(e.currentTarget.value)}
            error={
              hardCapCents === undefined
                ? "Enter a whole dollar amount, or leave empty."
                : softCapCents !== null &&
                    softCapCents !== undefined &&
                    hardCapCents !== null &&
                    softCapCents > hardCapCents
                  ? "The hard cap must be at least the soft cap."
                  : undefined
            }
          />
          <p className="text-[12px] text-gray-10">
            Caps are stored with your billing settings today; notifications and enforcement ship
            next.
          </p>
        </div>
      </DialogContainer>
    </>
  );
};
