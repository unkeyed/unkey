"use client";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import { IconCircleCheckOutline18 } from "@unkey/icons";
import { cn } from "cn";
import { EnvironmentLabel } from "../../../components/environment-label";
import { CUSTOM_DOMAIN_COLUMNS } from "./custom-domain-row";

type PlatformDomainRowProps = {
  domain: Domain;
  environment?: Environment;
};

export function PlatformDomainRow({ domain, environment }: PlatformDomainRowProps) {
  const hostname = domain.fullyQualifiedDomainName;

  return (
    <div className={cn(CUSTOM_DOMAIN_COLUMNS, "h-12 px-4 text-xs")}>
      <span className="flex size-6 items-center justify-center">
        <IconCircleCheckOutline18 className="size-4 text-success-11" />
      </span>
      <span className="flex min-w-0">
        <a
          href={`https://${hostname}`}
          target="_blank"
          rel="noopener noreferrer"
          className="truncate text-sm font-medium text-gray-12 hover:underline"
        >
          {hostname}
        </a>
      </span>
      <span />
      <span className="flex min-w-0 items-center">
        <EnvironmentLabel environment={environment} className="text-xs" />
      </span>
    </div>
  );
}
