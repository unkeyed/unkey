"use client";

import { Ban, BracketsCurly, CircleHalfDottedClock, Key, Lock, ShieldAlert } from "@unkey/icons";
import { Dialog, DialogContent } from "@unkey/ui";
import type { PolicyType } from "./types";

const POLICY_TYPES: {
  type: PolicyType;
  label: string;
  description: string;
  icon: React.ReactNode;
}[] = [
  {
    type: "keyauth",
    label: "Key Auth",
    description: "Authenticate requests using Unkey API keys",
    icon: <Key className="text-gray-11" />,
  },
  {
    type: "jwtauth",
    label: "JWT Auth",
    description: "Validate Bearer JWTs via JWKS / OIDC",
    icon: <Lock className="text-gray-11" />,
  },
  {
    type: "basicauth",
    label: "Basic Auth",
    description: "HTTP Basic authentication with static credentials",
    icon: <ShieldAlert className="text-gray-11" />,
  },
  {
    type: "ratelimit",
    label: "Rate Limit",
    description: "Enforce request rate limits per key source",
    icon: <CircleHalfDottedClock className="text-gray-11" />,
  },
  {
    type: "ipRules",
    label: "IP Rules",
    description: "Allow or deny requests by CIDR ranges",
    icon: <Ban className="text-gray-11" />,
  },
  {
    type: "openapi",
    label: "OpenAPI Validation",
    description: "Validate requests against an OpenAPI spec",
    icon: <BracketsCurly className="text-gray-11" />,
  },
];

export function AddPolicyDialog({
  open,
  onOpenChange,
  onSelect,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: (type: PolicyType) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <h3 className="text-base font-semibold text-gray-12 mb-3">Add Policy</h3>
        <div className="flex flex-col gap-1">
          {POLICY_TYPES.map((pt) => (
            <button
              key={pt.type}
              type="button"
              className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-left hover:bg-grayA-3 transition-colors"
              onClick={() => onSelect(pt.type)}
            >
              {pt.icon}
              <div>
                <p className="text-sm font-medium text-gray-12">{pt.label}</p>
                <p className="text-xs text-gray-11">{pt.description}</p>
              </div>
            </button>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
