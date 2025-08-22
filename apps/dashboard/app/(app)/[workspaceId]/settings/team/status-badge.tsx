"use client";

import { Badge } from "@unkey/ui";
import { memo } from "react";

type StatusBadgeProps = {
  status: "pending" | "accepted" | "revoked" | "expired";
};

export const StatusBadge = memo(({ status }: StatusBadgeProps) => {
  switch (status) {
    case "pending":
      return <Badge variant="primary">Pending</Badge>;
    case "accepted":
      return <Badge variant="secondary">Accepted</Badge>;
    case "revoked":
      return <Badge variant="secondary">Revoked</Badge>;
    case "expired":
      return <Badge variant="secondary">Expired</Badge>;
    default:
      return null;
  }
});

StatusBadge.displayName = "StatusBadge";
