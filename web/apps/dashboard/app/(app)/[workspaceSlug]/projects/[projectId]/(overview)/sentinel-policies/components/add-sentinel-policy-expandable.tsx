"use client";

import type { ComboboxOption } from "@/components/ui/combobox";
import { FormCombobox } from "@/components/ui/form-combobox";
import { trpc } from "@/lib/trpc/client";
import type {
  MatchCondition,
  SentinelPolicy,
  StringMatchMode,
} from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { ChevronDown, DoubleChevronRight, Plus, Trash, XMark } from "@unkey/icons";
import {
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SlidePanel,
} from "@unkey/ui";
import { useState } from "react";

type SentinelPolicyAddPanelProps = {
  envASlug: string;
  envBSlug: string;
  isOpen: boolean;
  topOffset: number;
  onClose: () => void;
  onAdd: (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => void;
};

const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

const STRING_MATCH_MODES: { value: StringMatchMode; label: string }[] = [
  { value: "exact", label: "Exact" },
  { value: "prefix", label: "Prefix" },
  { value: "regex", label: "Regex" },
];

const MATCH_TYPE_OPTIONS: { value: MatchCondition["type"]; label: string }[] = [
  { value: "path", label: "Path" },
  { value: "method", label: "Method" },
  { value: "header", label: "Header" },
  { value: "queryParam", label: "Query Param" },
];

function MatchConditionCard({
  condition,
  onChange,
  onDelete,
}: {
  condition: MatchCondition;
  onChange: (updated: MatchCondition) => void;
  onDelete: (id: string) => void;
}) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <div className="flex-1">
          <Select
            value={condition.type}
            onValueChange={(v) => {
              const type = v as MatchCondition["type"];
              const base = { id: condition.id };
              const reset: MatchCondition =
                type === "path"
                  ? { ...base, type: "path", mode: "exact", value: "" }
                  : type === "method"
                    ? { ...base, type: "method", methods: [] }
                    : type === "header"
                      ? { ...base, type: "header", name: "" }
                      : { ...base, type: "queryParam", name: "" };
              onChange(reset);
            }}
          >
            <SelectTrigger
              rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="z-60">
              {MATCH_TYPE_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="size-9 shrink-0 px-0 justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 rounded-lg"
          onClick={() => onDelete(condition.id)}
        >
          <Trash iconSize="sm-regular" />
        </Button>
      </div>

      {condition.type === "path" && (
        <div className="flex gap-2">
          <div className="w-28 shrink-0">
            <Select
              value={condition.mode}
              onValueChange={(v) => onChange({ ...condition, mode: v as StringMatchMode })}
            >
              <SelectTrigger
                rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="z-60">
                {STRING_MATCH_MODES.map((m) => (
                  <SelectItem key={m.value} value={m.value}>
                    {m.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <FormInput
            placeholder={condition.mode === "regex" ? "^/api/.*" : "/api/v1"}
            value={condition.value}
            onChange={(e) => onChange({ ...condition, value: e.target.value })}
            className="flex-1"
          />
        </div>
      )}

      {condition.type === "method" && (
        <div className="flex flex-wrap gap-1.5">
          {HTTP_METHODS.map((m) => {
            const active = condition.methods.includes(m);
            return (
              <button
                key={m}
                type="button"
                onClick={() =>
                  onChange({
                    ...condition,
                    methods: active
                      ? condition.methods.filter((x) => x !== m)
                      : [...condition.methods, m],
                  })
                }
                className={`px-2 py-0.5 rounded text-xs font-mono border transition-colors ${
                  active
                    ? "bg-accent-3 border-accent-6 text-accent-11"
                    : "bg-grayA-2 border-grayA-4 text-grayA-9 hover:text-gray-12"
                }`}
              >
                {m}
              </button>
            );
          })}
        </div>
      )}

      {(condition.type === "header" || condition.type === "queryParam") && (
        <div className="flex flex-col gap-2">
          <FormInput
            placeholder={condition.type === "header" ? "X-Custom-Header" : "param_name"}
            value={condition.name}
            onChange={(e) => onChange({ ...condition, name: e.target.value })}
          />
          {!condition.present && (
            <div className="flex gap-2">
              <div className="w-28 shrink-0">
                <Select
                  value={condition.mode ?? "exact"}
                  onValueChange={(v) => onChange({ ...condition, mode: v as StringMatchMode })}
                >
                  <SelectTrigger
                    rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="z-60">
                    {STRING_MATCH_MODES.map((m) => (
                      <SelectItem key={m.value} value={m.value}>
                        {m.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <FormInput
                placeholder="value"
                value={condition.value ?? ""}
                onChange={(e) => onChange({ ...condition, value: e.target.value })}
                className="flex-1"
              />
            </div>
          )}
        </div>
      )}
    </div>
  );
}

const POLICY_TYPE_OPTIONS: { value: SentinelPolicy["type"]; label: string }[] = [
  { value: "keyauth", label: "Key Auth" },
  { value: "ratelimit", label: "Rate Limit" },
  { value: "jwt", label: "JWT Auth" },
  { value: "basicauth", label: "Basic Auth" },
  { value: "iprules", label: "IP Rules" },
  { value: "openapi", label: "OpenAPI" },
];

export function SentinelPolicyAddPanel({
  envASlug,
  envBSlug,
  isOpen,
  topOffset,
  onClose,
  onAdd,
}: SentinelPolicyAddPanelProps) {
  const [name, setName] = useState("");
  const [type, setType] = useState<SentinelPolicy["type"]>("ratelimit");
  const [environmentId, setEnvironmentId] = useState<"__all__" | string>("__all__");
  const [matchConditions, setMatchConditions] = useState<MatchCondition[]>([]);

  // Ratelimit
  const [limit, setLimit] = useState(100);
  const [windowMs, setWindowMs] = useState(60000);

  // Keyauth
  const [keySpaceIds, setKeySpaceIds] = useState<string[]>([]);

  const { data: availableKeyspaces = {} } =
    trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery(undefined, {
      enabled: type === "keyauth" && isOpen,
    });

  const handleTypeChange = (newType: SentinelPolicy["type"]) => {
    setType(newType);
    // Reset config state on type change
    setLimit(100);
    setWindowMs(60000);
    setKeySpaceIds([]);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const configPatch: Partial<SentinelPolicy> =
      type === "ratelimit"
        ? { ratelimit: { limit, windowMs } }
        : type === "keyauth"
          ? { keyauth: { keySpaceIds } }
          : {};

    const id = crypto.randomUUID();
    const matchPatch = matchConditions.length > 0 ? { match: { conditions: matchConditions } } : {};
    const base: SentinelPolicy = { id, name, enabled: true, type, ...configPatch, ...matchPatch };

    const prodPolicy = environmentId === "__all__" || environmentId === envASlug ? base : null;
    const previewPolicy =
      environmentId === "__all__" || environmentId === envBSlug
        ? { ...base, enabled: environmentId === envBSlug }
        : null;

    onAdd(prodPolicy, previewPolicy);
    onClose();
    // Reset form
    setName("");
    setType("ratelimit");
    setLimit(100);
    setWindowMs(60000);
    setKeySpaceIds([]);
    setEnvironmentId("__all__");
    setMatchConditions([]);
  };

  const unselected = Object.keys(availableKeyspaces).filter((id) => !keySpaceIds.includes(id));
  const comboboxOptions: ComboboxOption[] = unselected.map((id) => ({
    value: id,
    searchValue: id,
    label: (
      <span className="text-gray-11 text-xs font-mono">
        {availableKeyspaces[id]?.api?.name ?? id}
      </span>
    ),
  }));

  return (
    <SlidePanel.Root isOpen={isOpen} onClose={onClose} topOffset={topOffset}>
      <SlidePanel.Header>
        <div className="flex flex-col">
          <span className="text-gray-12 font-medium text-base leading-8">Add Policy</span>
          <span className="text-gray-11 text-[13px] leading-5">
            Configure and add a new sentinel policy.
          </span>
        </div>
        <SlidePanel.Close
          aria-label="Close panel"
          className="mt-0.5 inline-flex items-center justify-center size-9 rounded-md hover:bg-grayA-3 transition-colors cursor-pointer"
        >
          <DoubleChevronRight
            iconSize="lg-medium"
            className="text-gray-10 transition-transform duration-300 ease-out group-hover:text-gray-12"
          />
        </SlidePanel.Close>
      </SlidePanel.Header>

      <SlidePanel.Content>
        <form onSubmit={handleSubmit} className="h-full flex flex-col">
          <div className="flex-1 overflow-y-auto pt-6 bg-grayA-2">
            <div className="flex flex-col gap-5 px-8">
              <FormInput
                label="Name"
                placeholder="Policy name"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />

              <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
                <label className="text-gray-11 text-[13px]">Type</label>
                <Select
                  value={type}
                  onValueChange={(v) => handleTypeChange(v as SentinelPolicy["type"])}
                >
                  <SelectTrigger
                    className="capitalize"
                    rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="z-60">
                    {POLICY_TYPE_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </fieldset>

              {type === "ratelimit" && (
                <div className="flex gap-3">
                  <FormInput
                    label="Limit"
                    type="number"
                    value={limit}
                    onChange={(e) => setLimit(Number.parseInt(e.target.value) || 0)}
                    className="flex-1"
                  />
                  <FormInput
                    label="Window (ms)"
                    type="number"
                    value={windowMs}
                    onChange={(e) => setWindowMs(Number.parseInt(e.target.value) || 0)}
                    className="flex-1"
                    description={windowMs > 0 ? `${(windowMs / 1000).toFixed(1)}s` : ""}
                  />
                </div>
              )}

              {type === "keyauth" && (
                <FormCombobox
                  label="Keyspaces"
                  options={comboboxOptions}
                  value=""
                  onSelect={(id) => {
                    if (!keySpaceIds.includes(id)) {
                      setKeySpaceIds((prev) => [...prev, id]);
                    }
                  }}
                  placeholder={
                    keySpaceIds.length === 0 ? (
                      <span className="text-grayA-8 w-full text-left">Select a keyspace</span>
                    ) : (
                      <div className="w-full flex flex-wrap gap-1.5 py-0.5">
                        {keySpaceIds.map((id) => (
                          <span
                            key={id}
                            className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-grayA-3 border border-grayA-4 text-xs text-accent-12"
                          >
                            {availableKeyspaces[id]?.api?.name ?? id}
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                setKeySpaceIds((prev) => prev.filter((k) => k !== id));
                              }}
                              className="p-0.5 hover:bg-grayA-4 rounded text-grayA-9 hover:text-accent-12 transition-colors"
                            >
                              <XMark iconSize="sm-regular" />
                            </button>
                          </span>
                        ))}
                      </div>
                    )
                  }
                  searchPlaceholder="Search keyspaces..."
                  emptyMessage={<div className="mt-2">No keyspaces available.</div>}
                />
              )}
            </div>
          </div>

          <div className="border-t border-grayA-4">
            <div className="px-8 pt-6">
              <label className="text-gray-11 text-[13px]">Match Conditions</label>
            </div>

            {matchConditions.length === 0 ? (
              <p className="text-[13px] text-grayA-8 px-8 pt-3">Applies to all requests</p>
            ) : (
              <div className="flex flex-col gap-4 px-8 pt-3 max-h-[400px] overflow-y-auto">
                {matchConditions.map((cond) => (
                  <MatchConditionCard
                    key={cond.id}
                    condition={cond}
                    onChange={(updated) =>
                      setMatchConditions((prev) =>
                        prev.map((c) => (c.id === updated.id ? updated : c)),
                      )
                    }
                    onDelete={(id) => setMatchConditions((prev) => prev.filter((c) => c.id !== id))}
                  />
                ))}
              </div>
            )}

            <div className="flex py-6 px-8">
              <Button
                type="button"
                variant="outline"
                size="md"
                className="font-medium"
                onClick={() =>
                  setMatchConditions((prev) => [
                    ...prev,
                    { id: crypto.randomUUID(), type: "path", mode: "exact", value: "" },
                  ])
                }
              >
                <Plus iconSize="sm-regular" />
                Add Condition
              </Button>
            </div>
          </div>

          <div className="border-t border-grayA-4">
            <div className="px-8 py-6">
              <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
                <label className="text-gray-11 text-[13px]">Environment</label>
                <Select value={environmentId} onValueChange={setEnvironmentId}>
                  <SelectTrigger
                    className="capitalize"
                    rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="z-60">
                    <SelectItem value="__all__">All Environments</SelectItem>
                    <SelectItem value={envASlug} className="capitalize">
                      {envASlug}
                    </SelectItem>
                    <SelectItem value={envBSlug} className="capitalize">
                      {envBSlug}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </fieldset>
            </div>
          </div>

          <div className="border-t border-gray-4 bg-white dark:bg-black px-8 py-5 flex items-center justify-end">
            <Button type="submit" variant="primary" size="md" className="px-3">
              Add Policy
            </Button>
          </div>
        </form>
      </SlidePanel.Content>
    </SlidePanel.Root>
  );
}
