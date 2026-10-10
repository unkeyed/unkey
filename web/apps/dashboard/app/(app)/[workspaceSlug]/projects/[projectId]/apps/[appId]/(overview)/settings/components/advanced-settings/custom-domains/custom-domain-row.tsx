"use client";
import { collection } from "@/lib/collections";
import {
  type CustomDomain,
  type VerificationStatus,
  retryDomainVerification,
} from "@/lib/collections/deploy/custom-domains";
import {
  IconCircleCheckOutline12,
  IconCircleInfoOutline18,
  IconClockOutline12,
  IconRefresh3Outline12,
  IconRefresh3Outline18,
  IconTriangleWarningOutline12,
} from "@unkey/icons";
import { Badge, Button, ConfirmPopover, Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { cn } from "cn";
import { useRef, useState } from "react";
import { RemoveButton } from "../../../../../../_components/remove-button";
import { ProviderIcon } from "../../../../domains/custom-domains/provider-icon";
import { DnsRecordTable } from "./dns-record-table";

type CustomDomainRowProps = {
  domain: CustomDomain;
  environmentSlug?: string;
};

const statusConfig: Record<
  VerificationStatus,
  { label: string; color: "primary" | "success" | "warning" | "error"; icon: React.ReactNode }
> = {
  pending: {
    label: "Pending",
    color: "primary",
    icon: <IconClockOutline12 className="!" />,
  },
  verifying: {
    label: "Verifying",
    color: "warning",
    icon: <IconRefresh3Outline12 className="! animate-spin" />,
  },
  verified: {
    label: "Verified",
    color: "success",
    icon: <IconCircleCheckOutline12 className="!" />,
  },
  failed: {
    label: "Failed",
    color: "error",
    icon: <IconTriangleWarningOutline12 className="!" />,
  },
};

export function CustomDomainRow({ domain, environmentSlug }: CustomDomainRowProps) {
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [isRetrying, setIsRetrying] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const status = statusConfig[domain.verificationStatus];

  const handleDelete = () => {
    collection.customDomains.delete(domain.id);
  };

  const handleRetry = async () => {
    setIsRetrying(true);
    try {
      await retryDomainVerification({ domain: domain.domain });
    } finally {
      setIsRetrying(false);
    }
  };

  return (
    <div className="border-b last:border-b-0 group">
      <div className="flex items-center justify-between px-4 py-3 h-12">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <a
            href={`https://${domain.domain}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-gray-12 font-medium hover:underline truncate"
          >
            {domain.domain}
          </a>
          {environmentSlug && (
            <Badge variant="secondary" size="sm" font="mono" className="shrink-0">
              {environmentSlug}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-2">
          <Badge variant={status.color} className="gap-1">
            {status.icon}
            {status.label}
          </Badge>

          {domain.verificationStatus === "failed" && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    size="icon"
                    variant="outline"
                    onClick={handleRetry}
                    disabled={isRetrying}
                    className="size-7 text-gray-9 hover:text-gray-11"
                  >
                    <IconRefresh3Outline18
                      className={cn("size-[14px]!", isRetrying && "animate-spin")}
                    />
                  </Button>
                }
              />
              <TooltipContent>Retry verification</TooltipContent>
            </Tooltip>
          )}

          {domain.verificationError && (
            <Tooltip>
              <TooltipTrigger>
                <IconCircleInfoOutline18 className="size-4! text-error-9" />
              </TooltipTrigger>
              <TooltipContent className="max-w-xs">{domain.verificationError}</TooltipContent>
            </Tooltip>
          )}

          <RemoveButton
            label={`Delete ${domain.domain}`}
            onClick={() => setIsConfirmOpen(true)}
            ref={deleteButtonRef}
            className="size-5.5"
          />

          {deleteButtonRef.current && (
            <ConfirmPopover
              isOpen={isConfirmOpen}
              onOpenChange={setIsConfirmOpen}
              triggerRef={deleteButtonRef}
              title="Delete domain"
              description={`Are you sure you want to delete ${domain.domain}? This will remove the domain and any associated routing.`}
              onConfirm={handleDelete}
              confirmButtonText="Delete"
              variant="danger"
            />
          )}
        </div>
      </div>

      {domain.verificationStatus !== "verified" &&
        domain.domainConnectUrl &&
        domain.domainConnectProvider && (
          <div className="mx-4 mb-3 flex items-center gap-3 px-4 py-3 rounded-lg border bg-raised">
            <ProviderIcon provider={domain.domainConnectProvider} className="size-6!" />
            <div className="flex-1">
              <p className="text-sm font-medium text-gray-12">Automatic setup available</p>
              <p className="text-xs text-gray-9">
                We detected your domain uses {domain.domainConnectProvider}. We can configure your
                DNS records automatically.
              </p>
            </div>
            <Button
              variant="primary"
              onClick={() =>
                window.open(domain.domainConnectUrl ?? undefined, "_blank", "noopener,noreferrer")
              }
            >
              Connect
            </Button>
          </div>
        )}

      {domain.verificationStatus !== "verified" && (
        <DnsRecordTable records={domain.dnsRecords} isLoading={domain.dnsRecords.length === 0} />
      )}
    </div>
  );
}
