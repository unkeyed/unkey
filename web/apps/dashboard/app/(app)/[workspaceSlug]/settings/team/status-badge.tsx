"use client";

import { Badge } from "@unkey/ui";
import { memo } from "react";

type StatusBadgeProps = {
  status: "pending" | "accepted" | "revoked" | "expired";
};

export const StatusBadge = memo(({ status }: StatusBadgeProps) => {
  switch (status) {
    case "pending":
      return (
        <Badge variant="warning" className="text-xs">
          Pending
        </Badge>
      );
    case "accepted":
      return (
        <Badge variant="success" className="text-xs">
          Accepted
        </Badge>
      );
    case "revoked":
      return (
        <Badge variant="error" className="text-xs">
          Revoked
        </Badge>
      );
    case "expired":
      return (
        <Badge variant="error" className="text-xs">
          Expired
        </Badge>
      );
    default:
      return null;
  }
});

StatusBadge.displayName = "StatusBadge";
