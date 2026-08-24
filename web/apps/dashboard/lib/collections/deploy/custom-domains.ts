"use client";
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
  // The records to publish, already resolved by the API: apex domains get ALIAS
  // where subdomains get CNAME, and each carries its own verified flag.
  dnsRecords: z.array(dnsRecordSchema),
  verificationError: z.string().nullable(),
  domainConnectProvider: z.string().nullable(),
  domainConnectUrl: z.string().nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
});

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
      const { changes } = transaction.mutations[0];

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
          // The banner in the card carries the details and the actions.
          if (isCustomDomainLimitError(err)) {
            return { message: "Custom domain limit reached" };
          }
          if (isDomainConflictError(err)) {
            return {
              message: "Domain already in use",
              description: getErrorMessage(err),
              action: {
                label: "View",
                // Resolved on click: the owning app is only worth a request once
                // the user asks to see it.
                onClick: async () => {
                  const route = await openOwningApp(createInput.domain);
                  if (route) {
                    window.open(route, "_blank", "noopener,noreferrer");
                  }
                },
              },
            };
          }
          return getErrorToast(err, "Failed to add domain");
        },
      });

      const result = await mutation;
      transaction.metadata = { domainId: result.data.domainId };
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
      // A verified custom domain also serves traffic, so its removal changes the
      // platform domain set.
      await domains.utils.refetch();
    },
  }),
);

/**
 * The plan allowance gate answers 403, which the generic mapping would report as
 * a permission problem. The type URN separates it from an RBAC denial.
 */
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

function openOwningApp(domain: string): Promise<string | null> {
  return trpcClient.deploy.customDomain.ownerRoute.query({ domain }).catch(() => null);
}

function dnsSetupHint(records: DnsRecord[]): string {
  const routing = records.find((record) => record.type !== "TXT");
  return routing
    ? `Add a ${routing.type} record pointing to ${routing.value}`
    : "Add the DNS records shown below";
}

/**
 * The API lists the domains of one environment, so this walks every environment
 * of the project. Callers filter by app in their live query.
 */
async function listProjectDomains(projectId: string): Promise<CustomDomain[]> {
  const environments = await trpcClient.deploy.environment.list.query({ projectId });

  const [perEnvironment, hints] = await Promise.all([
    Promise.all(
      environments.map(async (environment) => {
        const domains = await listAllDomains(projectId, environment.appId, environment.id);
        return domains.map((domain) => toCustomDomain(domain));
      }),
    ),
    // Absent from the Domain object, so the row would lose its one-click setup
    // shortcut on a reload without this.
    trpcClient.deploy.customDomain.hints
      .query({ projectId })
      .catch(() => []),
  ]);

  const byDomain = new Map(hints.map((hint) => [hint.domain, hint]));

  return perEnvironment.flat().map((domain) => {
    const hint = byDomain.get(domain.domain);
    return hint
      ? { ...domain, domainConnectProvider: hint.provider, domainConnectUrl: hint.url }
      : domain;
  });
}

async function listAllDomains(
  projectId: string,
  appId: string,
  environmentId: string,
): Promise<ApiDomain[]> {
  const all: ApiDomain[] = [];
  let cursor: string | undefined;

  do {
    const page = await getUnkeyClient().domains.listDomains({
      project: projectId,
      app: appId,
      environment: environmentId,
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
