"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { parseProjectIdFromWhere, validateProjectIdInQuery } from "./utils";

const verificationStatusSchema = z.enum(["pending", "verifying", "verified", "failed"]);

const schema = z.object({
  id: z.string(),
  domain: z.string(),
  workspaceId: z.string(),
  projectId: z.string(),
  environmentId: z.string(),
  verificationStatus: verificationStatusSchema,
  verificationToken: z.string(),
  ownershipVerified: z.boolean(),
  cnameVerified: z.boolean(),
  targetCname: z.string(),
  checkAttempts: z.number(),
  lastCheckedAt: z.number().nullable(),
  verificationError: z.string().nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
});

export type CustomDomain = z.infer<typeof schema>;
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
    refetchInterval: 5000,
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

      return trpcClient.deploy.customDomain.list.query({ projectId });
    },
    getKey: (item) => item.id,
    id: "customDomains",
    onInsert: async ({ transaction }) => {
      const { changes } = transaction.mutations[0];

      const addInput = z
        .object({
          projectId: z.string().min(1),
          environmentId: z.string().min(1),
          domain: z.string().min(1),
        })
        .parse({
          projectId: changes.projectId,
          environmentId: changes.environmentId,
          domain: changes.domain,
        });

      const mutation = trpcClient.deploy.customDomain.add.mutate(addInput);

      toast.promise(mutation, {
        loading: "Adding domain...",
        success: (data) => ({
          message: "Domain added",
          description: `Add a CNAME record pointing to ${data.targetCname}`,
        }),
        error: (err) => ({
          message: "Failed to add domain",
          description: err.message,
        }),
      });

      await mutation;
    },
    onDelete: async ({ transaction }) => {
      const original = transaction.mutations[0].original;

      const deleteMutation = trpcClient.deploy.customDomain.delete.mutate({
        domain: original.domain,
        projectId: original.projectId,
      });

      toast.promise(deleteMutation, {
        loading: "Deleting domain...",
        success: "Domain deleted",
        error: (err) => ({
          message: "Failed to delete domain",
          description: err.message,
        }),
      });

      await deleteMutation;
    },
  }),
);
