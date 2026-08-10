"use client";

import { Cube, Nodes } from "@unkey/icons";
import { ProductCard } from "./product-card";
import { SpendManagement } from "./spend-management";

type CostControlProps = {
  /** Month-to-date gross Compute usage spend in cents, or null while loading. */
  usageCents: number | null;
  isAdmin: boolean;
  /** False when Deploy billing is not configured server-side, as on the plans card. */
  showCompute: boolean;
};

/**
 * Split per product so the Compute cap cannot be read as workspace-wide: API
 * management sitting beside it with a fixed fee is what scopes it. The API card
 * says why it has no cap and stops there — what happens past the request quota
 * is enforcement policy, and it is due to change.
 */
export function CostControl({ usageCents, isAdmin, showCompute }: CostControlProps) {
  return (
    <>
      {showCompute ? (
        <ProductCard
          icon={<Cube iconSize="md-regular" />}
          iconClassName="bg-orangeA-3 text-orange-11"
          className="[&>div:nth-child(2)]:border-t-0 [&>div:nth-child(2)]:pt-0"
          name="Compute"
          subtitle="Usage is metered, so a monthly budget is what bounds the spend."
        >
          <SpendManagement usageCents={usageCents} isAdmin={isAdmin} />
        </ProductCard>
      ) : null}
      <ProductCard
        icon={<Nodes iconSize="md-regular" />}
        iconClassName="bg-infoA-3 text-info-11"
        name="API management"
        subtitle="A fixed monthly fee, so there is no budget to set."
      >
        <p className="text-[13px] text-gray-11 leading-5">
          API management bills a flat monthly fee for an included request quota. The requests are
          bought with the plan and carry no per-request charge, so this product's spend cannot rise
          with use.
        </p>
      </ProductCard>
    </>
  );
}
