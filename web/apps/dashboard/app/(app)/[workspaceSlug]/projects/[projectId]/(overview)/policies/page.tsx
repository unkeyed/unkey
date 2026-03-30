"use client";

import { ConfigSchema } from "@/gen/proto/config/v1/config_pb";
import { collection } from "@/lib/collections";
import { useSettingsIsSaving } from "@/lib/collections/deploy/environment-settings";
import { create, fromJson, toJson } from "@bufbuild/protobuf";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Plus, ShieldCheck } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { Reorder } from "framer-motion";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { useProjectData } from "../data-provider";
import { AddPolicyDialog } from "./components/add-policy-dialog";
import { PolicyCard } from "./components/policy-card";
import { fromProtobufPolicy, toProtobufPolicy } from "./components/serialization";
import type { PolicyFormData, PolicyType } from "./components/types";

function useEnvironmentId() {
  const { environments } = useProjectData();
  const searchParams = new URLSearchParams(
    typeof window !== "undefined" ? window.location.search : "",
  );
  const envIdParam = searchParams.get("environmentId");
  if (envIdParam) {
    const match = environments.find((e) => e.id === envIdParam);
    if (match) {
      return match.id;
    }
  }
  return environments.find((e) => e.slug === "production")?.id ?? environments.at(0)?.id;
}

export default function PoliciesPage() {
  const environmentId = useEnvironmentId();
  const [policies, setPolicies] = useState<PolicyFormData[]>([]);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [hasLoaded, setHasLoaded] = useState(false);
  const isSaving = useSettingsIsSaving();

  const getEnvId = useCallback(() => environmentId ?? "", [environmentId]);

  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, getEnvId())),
    [getEnvId],
  );

  const settings = data.at(0);

  // Load policies from collection data
  const sentinelConfigJson = useMemo(
    () => (settings?.sentinelConfig ? JSON.stringify(settings.sentinelConfig) : null),
    [settings?.sentinelConfig],
  );

  useEffect(() => {
    if (!sentinelConfigJson || hasLoaded) {
      return;
    }
    try {
      const raw = JSON.parse(sentinelConfigJson);
      const config = fromJson(ConfigSchema, raw);
      setPolicies(config.policies.map(fromProtobufPolicy));
    } catch {
      setPolicies([]);
    }
    setHasLoaded(true);
  }, [sentinelConfigJson, hasLoaded]);

  const handleSave = useCallback(() => {
    if (!environmentId) {
      return;
    }
    const config = create(ConfigSchema, {
      policies: policies.map(toProtobufPolicy),
    });
    const json = toJson(ConfigSchema, config);

    collection.environmentSettings.update(environmentId, (draft) => {
      draft.sentinelConfig = json as Record<string, unknown>;
    });
  }, [environmentId, policies]);

  const addPolicy = (type: PolicyType) => {
    const id = crypto.randomUUID();
    const names: Record<PolicyType, string> = {
      keyauth: "Key Auth",
      jwtauth: "JWT Auth",
      basicauth: "Basic Auth",
      ratelimit: "Rate Limit",
      ipRules: "IP Rules",
      openapi: "OpenAPI Validation",
    };
    const newPolicy: PolicyFormData = {
      id,
      name: names[type],
      enabled: true,
      type,
      match: [],
      config: getDefaultConfig(type),
    };
    setPolicies((prev) => [...prev, newPolicy]);
    setShowAddDialog(false);
  };

  const updatePolicy = (id: string, updates: Partial<PolicyFormData>) => {
    setPolicies((prev) => prev.map((p) => (p.id === id ? { ...p, ...updates } : p)));
  };

  const removePolicy = (id: string) => {
    setPolicies((prev) => prev.filter((p) => p.id !== id));
  };

  return (
    <ProjectContentWrapper centered>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <ShieldCheck className="text-gray-12" />
          <div>
            <h2 className="text-lg font-semibold text-gray-12">Sentinel Policies</h2>
            <p className="text-sm text-gray-11">
              Configure middleware policies evaluated in order on each request.
            </p>
          </div>
        </div>
        <Button
          variant="primary"
          size="sm"
          onClick={handleSave}
          loading={isSaving}
          disabled={!environmentId}
        >
          Save
        </Button>
      </div>

      {policies.length === 0 ? (
        <div className="rounded-xl border border-grayA-4 bg-grayA-2 p-12 flex flex-col items-center gap-4 text-center">
          <ShieldCheck className="text-grayA-8" />
          <div>
            <p className="text-sm font-medium text-gray-12">No policies configured</p>
            <p className="text-sm text-gray-11 mt-1">Add a policy to start protecting your API.</p>
          </div>
          <Button variant="primary" size="sm" onClick={() => setShowAddDialog(true)}>
            <Plus className="size-3" />
            Add Policy
          </Button>
        </div>
      ) : (
        <>
          <Reorder.Group
            axis="y"
            values={policies}
            onReorder={setPolicies}
            className="flex flex-col gap-3"
          >
            {policies.map((policy) => (
              <Reorder.Item key={policy.id} value={policy} layout="position">
                <PolicyCard
                  policy={policy}
                  onUpdate={(updates) => updatePolicy(policy.id, updates)}
                  onRemove={() => removePolicy(policy.id)}
                />
              </Reorder.Item>
            ))}
          </Reorder.Group>
          <Button
            variant="ghost"
            size="sm"
            className="w-full border border-dashed border-grayA-4"
            onClick={() => setShowAddDialog(true)}
          >
            <Plus className="size-3" />
            Add Policy
          </Button>
        </>
      )}

      <AddPolicyDialog open={showAddDialog} onOpenChange={setShowAddDialog} onSelect={addPolicy} />
    </ProjectContentWrapper>
  );
}

function getDefaultConfig(type: PolicyType): PolicyFormData["config"] {
  switch (type) {
    case "keyauth":
      return { keySpaceIds: [], locations: [], permissionQuery: "" };
    case "jwtauth":
      return {
        jwksSource: "oidcIssuer",
        jwksValue: "",
        issuer: "",
        audiences: [],
        algorithms: [],
        subjectClaim: "sub",
        forwardClaims: [],
        allowAnonymous: false,
      };
    case "basicauth":
      return { credentials: [] };
    case "ratelimit":
      return { limit: 100, windowMs: 60000, keySource: "remoteIp", keyValue: "" };
    case "ipRules":
      return { allow: [], deny: [] };
    case "openapi":
      return { specYaml: "" };
  }
}
