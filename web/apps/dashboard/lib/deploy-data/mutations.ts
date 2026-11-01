"use client";

import { collection } from "@/lib/collections";
import type { App } from "@/lib/collections/deploy/apps";
import { retryDomainVerification } from "@/lib/collections/deploy/custom-domains";

type AppChanges = Partial<Pick<App, "name" | "defaultBranch" | "imageReference">>;

export function useAppMutations() {
  return APP_MUTATIONS;
}

export function useCustomDomainMutations() {
  return CUSTOM_DOMAIN_MUTATIONS;
}

const APP_MUTATIONS = {
  updateApp: (appId: string, changes: AppChanges): Promise<unknown> =>
    collection.apps.update(appId, { optimistic: false }, (draft) => {
      Object.assign(draft, changes);
    }).isPersisted.promise,
  deleteApp: (appId: string): void => {
    collection.apps.delete(appId);
  },
};

const CUSTOM_DOMAIN_MUTATIONS = {
  addCustomDomain: (
    domain: { domain: string; projectId: string; appId: string; environmentId: string },
    workspaceSlug: string,
  ): Promise<unknown> =>
    collection.customDomains.insert(
      {
        ...domain,
        id: crypto.randomUUID(),
        verificationStatus: "pending",
        dnsRecords: [],
        verificationError: null,
        domainConnectProvider: null,
        domainConnectUrl: null,
        createdAt: Date.now(),
        updatedAt: null,
      },
      { metadata: { workspaceSlug } },
    ).isPersisted.promise,
  deleteCustomDomain: (customDomainId: string): void => {
    collection.customDomains.delete(customDomainId);
  },
  retryDomainVerification,
};
