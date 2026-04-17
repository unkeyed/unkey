"use client";

import { shortenId } from "@/lib/shorten-id";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { Fingerprint } from "@unkey/icons";
import { CopyButton, InfoTooltip } from "@unkey/ui";
import Link from "next/link";
import type { z } from "zod";

type Identity = z.infer<typeof IdentityResponseSchema>;

type IdentityExternalIdCellProps = {
  identity: Identity;
  workspaceSlug: string;
};

export const IdentityExternalIdCell = ({
  identity,
  workspaceSlug,
}: IdentityExternalIdCellProps) => {
  const truncatedExternalId =
    identity.externalId.length > 50
      ? `${identity.externalId.slice(0, 50)}...`
      : identity.externalId;

  return (
    <div className="flex flex-col items-start px-[18px] py-[6px]">
      <div className="flex gap-4 items-center">
        <div className="size-5 rounded-sm flex items-center justify-center bg-brandA-3">
          <Fingerprint iconSize="md-medium" className="text-brandA-11" />
        </div>
        <div className="flex flex-col gap-1 text-xs">
          <span
            className="font-sans text-accent-12 font-medium truncate"
            title={identity.externalId}
          >
            {truncatedExternalId}
          </span>
          <InfoTooltip
            content={
              <div className="inline-flex justify-center gap-3 items-center font-mono text-xs text-gray-11">
                <span>{identity.id}</span>
                <CopyButton value={identity.id} />
              </div>
            }
            position={{ side: "bottom", align: "start" }}
          >
            <Link
              className="font-mono group-hover:underline decoration-dotted text-accent-9 w-full inline-block text-left"
              href={`/${workspaceSlug}/identities/${identity.id}`}
              onClick={(e) => e.stopPropagation()}
            >
              {shortenId(identity.id)}
            </Link>
          </InfoTooltip>
        </div>
      </div>
    </div>
  );
};
