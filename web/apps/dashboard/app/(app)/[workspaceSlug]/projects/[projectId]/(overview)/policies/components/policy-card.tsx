"use client";

import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { ChevronDown, Trash } from "@unkey/icons";
import { Badge, Button, FormInput } from "@unkey/ui";
import { useState } from "react";
import { BasicAuthForm } from "./forms/basicauth-form";
import { IPRulesForm } from "./forms/iprules-form";
import { JWTAuthForm } from "./forms/jwtauth-form";
import { KeyAuthForm } from "./forms/keyauth-form";
import { OpenAPIForm } from "./forms/openapi-form";
import { RateLimitForm } from "./forms/ratelimit-form";
import { MatchEditor } from "./match-editor";
import type {
  BasicAuthConfig,
  IPRulesConfig,
  JWTAuthConfig,
  KeyAuthConfig,
  OpenAPIConfig,
  PolicyFormData,
  PolicyType,
  RateLimitConfig,
} from "./types";

const TYPE_LABELS: Record<PolicyType, string> = {
  keyauth: "Key Auth",
  jwtauth: "JWT Auth",
  basicauth: "Basic Auth",
  ratelimit: "Rate Limit",
  ipRules: "IP Rules",
  openapi: "OpenAPI",
};

export function PolicyCard({
  policy,
  onUpdate,
  onRemove,
}: {
  policy: PolicyFormData;
  onUpdate: (updates: Partial<PolicyFormData>) => void;
  onRemove: () => void;
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="rounded-xl border border-grayA-4 bg-gray-1 overflow-hidden">
      {/* biome-ignore lint/a11y/useSemanticElements: can't use button — contains nested interactive elements */}
      <div
        role="button"
        tabIndex={0}
        className="flex items-center gap-3 px-4 py-3 cursor-pointer select-none"
        onClick={() => setExpanded(!expanded)}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            setExpanded(!expanded);
          }
        }}
      >
        {/* Drag handle */}
        {/* biome-ignore lint/a11y/useKeyWithClickEvents: drag handle is mouse-only by nature */}
        <div
          className="cursor-grab active:cursor-grabbing text-grayA-8 hover:text-grayA-11 shrink-0"
          title="Drag to reorder"
          onClick={(e) => e.stopPropagation()}
        >
          <svg width="10" height="14" viewBox="0 0 10 14" fill="currentColor">
            <circle cx="2" cy="2" r="1.5" />
            <circle cx="8" cy="2" r="1.5" />
            <circle cx="2" cy="7" r="1.5" />
            <circle cx="8" cy="7" r="1.5" />
            <circle cx="2" cy="12" r="1.5" />
            <circle cx="8" cy="12" r="1.5" />
          </svg>
        </div>

        {/* biome-ignore lint/a11y/useKeyWithClickEvents: stopPropagation wrapper for Switch */}
        <div onClick={(e) => e.stopPropagation()}>
          <Switch checked={policy.enabled} onCheckedChange={(enabled) => onUpdate({ enabled })} />
        </div>

        <div className="flex-1 flex items-center gap-2 min-w-0">
          <span
            className={cn(
              "text-sm font-medium truncate",
              policy.enabled ? "text-gray-12" : "text-gray-9",
            )}
          >
            {policy.name}
          </span>
          <Badge variant="secondary" className="text-[10px] shrink-0">
            {TYPE_LABELS[policy.type]}
          </Badge>
        </div>

        <Button
          variant="ghost"
          size="sm"
          className="text-grayA-8 hover:text-red-10 shrink-0"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
        >
          <Trash className="size-3.5" />
        </Button>

        <ChevronDown
          className={cn(
            "size-4 text-grayA-8 transition-transform shrink-0",
            expanded && "rotate-180",
          )}
        />
      </div>

      {expanded && (
        <div className="border-t border-grayA-4 bg-grayA-2 px-4 py-4 flex flex-col gap-5">
          <FormInput
            label="Policy Name"
            value={policy.name}
            onChange={(e) => onUpdate({ name: e.target.value })}
          />

          <MatchEditor matches={policy.match} onChange={(match) => onUpdate({ match })} />

          <div className="border-t border-grayA-4 pt-4">
            <p className="text-xs font-medium text-gray-11 mb-3 uppercase tracking-wider">
              {TYPE_LABELS[policy.type]} Configuration
            </p>
            <PolicyConfigForm policy={policy} onUpdate={onUpdate} />
          </div>
        </div>
      )}
    </div>
  );
}

function PolicyConfigForm({
  policy,
  onUpdate,
}: {
  policy: PolicyFormData;
  onUpdate: (updates: Partial<PolicyFormData>) => void;
}) {
  const updateConfig = (config: PolicyFormData["config"]) => onUpdate({ config });

  switch (policy.type) {
    case "keyauth":
      return <KeyAuthForm config={policy.config as KeyAuthConfig} onChange={updateConfig} />;
    case "jwtauth":
      return <JWTAuthForm config={policy.config as JWTAuthConfig} onChange={updateConfig} />;
    case "basicauth":
      return <BasicAuthForm config={policy.config as BasicAuthConfig} onChange={updateConfig} />;
    case "ratelimit":
      return <RateLimitForm config={policy.config as RateLimitConfig} onChange={updateConfig} />;
    case "ipRules":
      return <IPRulesForm config={policy.config as IPRulesConfig} onChange={updateConfig} />;
    case "openapi":
      return <OpenAPIForm config={policy.config as OpenAPIConfig} onChange={updateConfig} />;
  }
}
