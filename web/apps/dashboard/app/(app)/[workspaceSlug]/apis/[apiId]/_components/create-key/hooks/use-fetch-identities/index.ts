"use client";

import { trpc } from "@/lib/trpc/client";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { toast } from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";
import type { z } from "zod";

type Identity = z.infer<typeof IdentityResponseSchema>;

const MAX_IDENTITY_FETCH_LIMIT = 10;

export const useFetchIdentities = (limit = MAX_IDENTITY_FETCH_LIMIT) => {
  const trpcUtils = trpc.useUtils();
  const [page, setPage] = useState(1);
  const [accumulated, setAccumulated] = useState<Map<string, Identity>>(() => new Map());

  const { data, isLoading, isFetching } = trpc.identity.query.useQuery(
    { limit, page },
    {
      staleTime: Number.POSITIVE_INFINITY,
      keepPreviousData: true,
      onSuccess(result) {
        setAccumulated((prev) => {
          const next = new Map(prev);
          for (const identity of result.identities) {
            next.set(identity.id, identity);
          }
          return next;
        });
      },
      onError(err) {
        if (err.data?.code === "NOT_FOUND") {
          toast.error("Failed to Load Identities", {
            description:
              "We couldn't find any identities for this workspace. Please try again or contact support@unkey.com.",
          });
        } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
          toast.error("Server Error", {
            description:
              "We were unable to load identities. Please try again or contact support@unkey.com",
          });
        } else {
          toast.error("Failed to Load Identities", {
            description:
              err.message ||
              "An unexpected error occurred. Please try again or contact support@unkey.com",
            action: {
              label: "Contact Support",
              onClick: () => window.open("mailto:support@unkey.com", "_blank"),
            },
          });
        }
      },
    },
  );

  const identities = useMemo(() => Array.from(accumulated.values()), [accumulated]);

  const hasNextPage = data ? page < data.totalPages : false;
  const isFetchingNextPage = isFetching && page > 1;

  const loadMore = useCallback(() => {
    if (!isFetching && hasNextPage) {
      setPage((p) => p + 1);
    }
  }, [isFetching, hasNextPage]);

  const refresh = useCallback(() => {
    setPage(1);
    setAccumulated(new Map());
    trpcUtils.identity.query.invalidate();
  }, [trpcUtils.identity.query]);

  return {
    identities,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    loadMore,
    refresh,
  };
};
