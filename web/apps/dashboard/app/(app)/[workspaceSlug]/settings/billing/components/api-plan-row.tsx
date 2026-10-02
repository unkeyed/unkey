"use client";

import { currentApiProduct } from "@/lib/billing/api-plan";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { IconNodesOutline18 } from "@unkey/icons";
import { Item, ItemActions, ItemContent, ItemMedia, ItemTitle } from "@unkey/ui";
import { useState } from "react";
import { PlanName, PlanPrice, PlanRowAction } from "./plan-row";
import { PlansScreen } from "./plans-screen";

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

type ApiPlanRowProps = {
  isAdmin: boolean | undefined;
  emphasize: boolean;
  products: BillingInfo["products"];
  subscription?: BillingInfo["subscription"];
  currentProductId?: BillingInfo["currentProductId"];
};

export function ApiPlanRow({
  isAdmin,
  emphasize,
  products,
  subscription,
  currentProductId,
}: ApiPlanRowProps) {
  const [showPlanModal, setShowPlanModal] = useState(false);

  const currentProduct = currentApiProduct({ products, subscription, currentProductId });

  return (
    <>
      <Item>
        <ItemMedia className="bg-infoA-3 text-info-11">
          <IconNodesOutline18 />
        </ItemMedia>
        <ItemContent>
          <ItemTitle className="truncate">API management</ItemTitle>
        </ItemContent>
        <ItemActions className="gap-3">
          <PlanName>{currentProduct ? currentProduct.name : "Free"}</PlanName>
          <PlanPrice feeCents={(currentProduct?.dollar ?? 0) * 100} />
          <span className="flex w-20 justify-end">
            <PlanRowAction
              isAdmin={isAdmin}
              hasPlan={currentProduct !== undefined}
              emphasize={emphasize}
              onClick={() => setShowPlanModal(true)}
              chooseLabel="Upgrade"
            />
          </span>
        </ItemActions>
      </Item>

      <PlansScreen open={showPlanModal} onOpenChange={setShowPlanModal} reason="api-plan" />
    </>
  );
}
