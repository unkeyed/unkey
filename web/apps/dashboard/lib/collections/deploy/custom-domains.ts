"use client";
import { routes } from "@/lib/navigation/routes";
import { getErrorMessage, getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import {
  type QueryCollectionUtils,
  parseLoadSubsetOptions,
  queryCollectionOptions,
} from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import type { Domain as ApiDomain } from "@unkey/api/models/components";
import {
  BadRequestErrorResponse,
  ConflictErrorResponse,
  ForbiddenErrorResponse,
} from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient } from "../client";
import { domains } from "./domains";
import { extractStringFilter } from "./utils";

const verificationStatusSchema = z.enum(["pending", "verifying", "verified", "failed"]);

const dnsRecordSchema = z.object({
  type: z.enum(["CNAME", "ALIAS", "TXT"]),
  name: z.string(),
  value: z.string(),
  verified: z.boolean(),
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
export const customDomains = createCollection<
  CustomDomain,
  string,
  QueryCollectionUtils<CustomDomain, string>
>(
  queryCollectionOptions({
    queryClient,
    syncMode: "on-demand",
    queryKey: (opts) => {
      const { filters } = parseLoadSubsetOptions(opts);
      const projectId = extractStringFilter(filters, "projectId");
      return projectId ? ["customDomains", projectId] : ["customDomains"];
    },
    retry: 3,
    queryFn: async (ctx) => {
      const { filters } = parseLoadSubsetOptions(ctx.meta?.loadSubsetOptions);
      const projectId = extractStringFilter(filters, "projectId");

      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      return (await listAllDomains(projectId)).map(toCustomDomain);
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
        success: "Domain added",
        error: (err) => {
          // The page banner or the domain field shows the details.
          if (isCustomDomainLimitError(err)) {
            return { message: "Custom domain limit reached" };
          }
          if (isInvalidDomainError(err)) {
            return { message: "Invalid domain" };
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

/** The server's hostname rule also rejects public suffixes, which the client cannot know. */
export function isInvalidDomainError(error: unknown): boolean {
  return error instanceof BadRequestErrorResponse && error.error.type.endsWith("/invalid_input");
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

  return routes.projects.apps.domains({
    workspaceSlug,
    projectId: owner.data.projectId,
    appId: owner.data.appId,
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
      verified: record.verified,
    })),
    verificationError: domain.verificationError ?? null,
    domainConnectProvider: domain.domainConnect?.provider ?? null,
    domainConnectUrl: domain.domainConnect?.url ?? null,
    createdAt: domain.createdAt,
    updatedAt: domain.updatedAt ?? null,
  };
}
