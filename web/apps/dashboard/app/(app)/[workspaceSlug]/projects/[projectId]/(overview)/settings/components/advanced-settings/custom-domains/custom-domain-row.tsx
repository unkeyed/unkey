"use client";
import { collection } from "@/lib/collections";
import {
  type CustomDomain,
  type VerificationStatus,
  retryDomainVerification,
} from "@/lib/collections/deploy/custom-domains";
import { cn } from "@/lib/utils";
import { CircleCheck, CircleInfo, Clock, Refresh3, TriangleWarning } from "@unkey/icons";
import { Badge, Button, ConfirmPopover, Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { useRef, useState } from "react";
import { useProjectData } from "../../../../data-provider";
import { RemoveButton } from "../../shared/remove-button";
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
    icon: <Clock className="size-3!" iconSize="sm-regular" />,
  },
  verifying: {
    label: "Verifying",
    color: "warning",
    icon: <Refresh3 className="size-3! animate-spin" iconSize="sm-regular" />,
  },
  verified: {
    label: "Verified",
    color: "success",
    icon: <CircleCheck className="size-3!" iconSize="sm-regular" />,
  },
  failed: {
    label: "Failed",
    color: "error",
    icon: <TriangleWarning className="size-3!" iconSize="sm-regular" />,
  },
};

export function CustomDomainRow({ domain, environmentSlug }: CustomDomainRowProps) {
  const { projectId } = useProjectData();
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
      await retryDomainVerification({ domain: domain.domain, projectId });
    } finally {
      setIsRetrying(false);
    }
  };

  return (
    <div className="border-b border-gray-4 last:border-b-0 group">
      <div className="flex items-center justify-between px-4 py-3 h-12">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <a
            href={`https://${domain.domain}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-[13px] text-gray-12 font-medium hover:underline truncate"
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
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="outline"
                  onClick={handleRetry}
                  disabled={isRetrying}
                  className="size-7 text-gray-9 hover:text-gray-11"
                >
                  <Refresh3 className={cn("size-[14px]!", isRetrying && "animate-spin")} />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Retry verification</TooltipContent>
            </Tooltip>
          )}

          {domain.verificationError && (
            <Tooltip>
              <TooltipTrigger>
                <CircleInfo className="size-4! text-error-9" />
              </TooltipTrigger>
              <TooltipContent className="max-w-xs">{domain.verificationError}</TooltipContent>
            </Tooltip>
          )}

          <RemoveButton
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

      {domain.verificationStatus !== "verified" && (
        <DnsRecordTable
          domain={domain.domain}
          targetCname={domain.targetCname}
          verificationToken={domain.verificationToken}
          ownershipVerified={domain.ownershipVerified}
          cnameVerified={domain.cnameVerified}
        />
      )}
    </div>
  );
}
