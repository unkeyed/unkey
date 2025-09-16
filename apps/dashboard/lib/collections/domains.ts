"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "./client";

const schema = z.object({
  id: z.string(),
  domain: z.string(),
  type: z.enum(["custom", "wildcard"]),
  projectId: z.string().nullable(),
});

export type Domain = z.infer<typeof schema>;

export const domains = createCollection<Domain>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["domains"],
    retry: 3,
    queryFn: () => trpcClient.deploy.domain.list.query(),

    getKey: (item) => item.id,
    onInsert: async () => {
      throw new Error("Not implemented");
    },
    onDelete: async () => {
      throw new Error("Not implemented");
    },
  }),
);
