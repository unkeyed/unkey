"use client";
import { routes } from "@/lib/navigation/routes";
import { getErrorMessage, getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import type { Domain as ApiDomain, DnsRecord } from "@unkey/api/models/components";
import { ConflictErrorResponse, ForbiddenErrorResponse } from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { domains } from "./domains";
import { parseProjectIdFromWhere, validateProjectIdInQuery } from "./utils";

const verificationStatusSchema = z.enum(["pending", "verifying", "verified", "failed"]);

const dnsRecordSchema = z.object({
  type: z.enum(["CNAME", "ALIAS", "TXT"]),
  name: z.string(),
  value: z.string(),
  ttl: z.number(),
  verified: z.boolean(),
  note: z.string().nullable(),
});

const schema = z.object({
  id: z.string(),
  domain: z.string(),
  projectId: z.string(),
  appId: z.string(),
  environmentId: z.string(),
  verificationStatus: verificationStatusSchema,
  dnsRecords: z.array(dnsRecordSchema),
  verificationError: z.string().nullable(),
  domainConnectProvider: z.string().nullable(),
  domainConnectUrl: z.string().nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
});

const insertMetaSchema = z.object({ workspaceSlug: z.string().min(1) });

export type CustomDomain = z.infer<typeof schema>;
export type CustomDomainDnsRecord = z.infer<typeof dnsRecordSchema>;
export type VerificationStatus = z.infer<typeof verificationStatusSchema>;

/**
 * Custom domains collection.
 *
 * IMPORTANT: All queries MUST filter by projectId:
 * .where(({ customDomain }) => eq(customDomain.projectId, projectId))
 */
export const customDomains = createCollection<CustomDomain, string>(
  queryCollectionOptions({
    queryClient,
    syncMode: "on-demand",
    queryKey: (opts) => {
      const projectId = parseProjectIdFromWhere(opts.where);
      return projectId ? ["customDomains", projectId] : ["customDomains"];
    },
    retry: 3,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;

      validateProjectIdInQuery(options?.where);
      const projectId = parseProjectIdFromWhere(options?.where);

      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      return listProjectDomains(projectId);
    },
    getKey: (item) => item.id,
    id: "customDomains",
    onInsert: async ({ transaction }) => {
      const { changes, metadata } = transaction.mutations[0];
      const insertMeta = insertMetaSchema.safeParse(metadata);
      const createInput = z
        .object({
          project: z.string().min(1),
          app: z.string().min(1),
          environment: z.string().min(1),
          domain: z.string().min(1),
        })
        .parse({
          project: changes.projectId,
          app: changes.appId,
          environment: changes.environmentId,
          domain: changes.domain,
        });

      const mutation = getUnkeyClient().domains.createDomain(createInput);

      toast.promise(mutation, {
        loading: "Adding domain...",
        success: (data) => ({
          message: "Domain added",
          description: dnsSetupHint(data.data.dnsRecords),
          duration: 10_000,
        }),
        error: (err) => {
          // The banner in the card shows the details and the actions.
          if (isCustomDomainLimitError(err)) {
            return { message: "Custom domain limit reached" };
          }
          if (isDomainConflictError(err)) {
            return {
              message: "Domain already in use",
              description: getErrorMessage(err),
              ...(insertMeta.success && {
                action: {
                  label: "View",
                  onClick: async () => {
                    const route = await openOwningApp(
                      createInput.domain,
                      insertMeta.data.workspaceSlug,
                    );
                    if (route) {
                      window.open(route, "_blank", "noopener,noreferrer");
                    }
                  },
                },
              }),
            };
          }
          return getErrorToast(err, "Failed to add domain");
        },
      });

      await mutation;
      await customDomains.utils.refetch();
    },
    onDelete: async ({ transaction }) => {
      const original = transaction.mutations[0].original;

      const deleteMutation = getUnkeyClient().domains.deleteDomain({ domain: original.domain });

      toast.promise(deleteMutation, {
        loading: "Deleting domain...",
        success: "Domain deleted",
        error: (err) => getErrorToast(err, "Failed to delete domain"),
      });

      await deleteMutation;
      await domains.utils.refetch();
    },
  }),
);

export function isCustomDomainLimitError(error: unknown): boolean {
  return (
    error instanceof ForbiddenErrorResponse &&
    error.error.type.endsWith("/custom_domain_limit_exceeded")
  );
}

export async function retryDomainVerification({ domain }: { domain: string }): Promise<void> {
  const mutation = getUnkeyClient().domains.verifyDomain({ domain });

  toast.promise(mutation, {
    loading: "Retrying verification...",
    success: "Verification restarted",
    error: (err) => getErrorToast(err, "Failed to retry verification"),
  });

  await mutation;
  await customDomains.utils.refetch();
}

function isDomainConflictError(error: unknown): boolean {
  return (
    error instanceof ConflictErrorResponse && error.error.type.endsWith("/domain_already_exists")
  );
}

// getDomain resolves any domain in the workspace and 404s for the rest, so a
// rejection is the "not ours to link to" case.
async function openOwningApp(domain: string, workspaceSlug: string): Promise<string | null> {
  const owner = await getUnkeyClient()
    .domains.getDomain({ domain })
    .catch(() => null);

  if (!owner) {
    return null;
  }

  return routes.projects.apps.settings({
    workspaceSlug,
    projectId: owner.data.projectId,
    appId: owner.data.appId,
  });
}

function dnsSetupHint(records: DnsRecord[]): string {
  const routing = records.find((record) => record.type !== "TXT");
  return routing
    ? `Add a ${routing.type} record pointing to ${routing.value}`
    : "Add the DNS records shown below";
}

async function listProjectDomains(projectId: string): Promise<CustomDomain[]> {
  const [apiDomains, hints] = await Promise.all([
    listAllDomains(projectId),
    trpcClient.deploy.customDomain.hints.query({ projectId }).catch(() => []),
  ]);

  const byDomain = new Map(hints.map((hint) => [hint.domain, hint]));

  return apiDomains.map((apiDomain) => {
    const domain = toCustomDomain(apiDomain);
    const hint = byDomain.get(domain.domain);
    return hint
      ? { ...domain, domainConnectProvider: hint.provider, domainConnectUrl: hint.url }
      : domain;
  });
}

async function listAllDomains(projectId: string): Promise<ApiDomain[]> {
  const all: ApiDomain[] = [];
  let cursor: string | undefined;

  do {
    const page = await getUnkeyClient().domains.listDomains({
      project: projectId,
      cursor,
    });
    all.push(...page.data);
    cursor = page.pagination?.hasMore ? page.pagination.cursor : undefined;
  } while (cursor);

  return all;
}

function toCustomDomain(domain: ApiDomain): CustomDomain {
  return {
    id: domain.id,
    domain: domain.domain,
    projectId: domain.projectId,
    appId: domain.appId,
    environmentId: domain.environmentId,
    verificationStatus: domain.status,
    dnsRecords: domain.dnsRecords.map((record) => ({
      type: record.type,
      name: record.name,
      value: record.value,
      ttl: record.ttl,
      verified: record.verified,
      note: record.note ?? null,
    })),
    verificationError: domain.verificationError ?? null,
    domainConnectProvider: null,
    domainConnectUrl: null,
    createdAt: domain.createdAt,
    updatedAt: domain.updatedAt ?? null,
  };
}
