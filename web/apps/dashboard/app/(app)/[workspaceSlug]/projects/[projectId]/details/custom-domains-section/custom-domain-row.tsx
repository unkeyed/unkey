"use client";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import {
  CircleCheck,
  CircleInfo,
  Clock,
  Link4,
  Refresh3,
  Trash,
  TriangleWarning,
  XMark,
} from "@unkey/icons";
import {
  Badge,
  Button,
  ConfirmPopover,
  CopyButton,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  toast,
} from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
import type { CustomDomain, VerificationStatus } from "./types";

type CustomDomainRowProps = {
  domain: CustomDomain;
  projectId: string;
  onDelete: () => void;
  onRetry: () => void;
};

const statusConfig: Record<
  VerificationStatus,
  { label: string; color: "primary" | "success" | "warning" | "error"; icon: React.ReactNode }
> = {
  pending: {
    label: "Pending",
    color: "primary",
    icon: <Clock className="!size-3" />,
  },
  verifying: {
    label: "Verifying",
    color: "warning",
    icon: <Refresh3 className="!size-3 animate-spin" />,
  },
  verified: {
    label: "Verified",
    color: "success",
    icon: <CircleCheck className="!size-3" />,
  },
  failed: {
    label: "Failed",
    color: "error",
    icon: <TriangleWarning className="!size-3" />,
  },
};

export function CustomDomainRow({ domain, projectId, onDelete, onRetry }: CustomDomainRowProps) {
  const deleteMutation = trpc.deploy.customDomain.delete.useMutation();
  const retryMutation = trpc.deploy.customDomain.retry.useMutation();
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const verificationStatus = domain.verificationStatus as VerificationStatus;
  const status = statusConfig[verificationStatus] ?? statusConfig.pending;

  const handleDelete = async () => {
    const mutation = deleteMutation.mutateAsync({
      domain: domain.domain,
      projectId,
    });

    toast.promise(mutation, {
      loading: "Deleting domain...",
      success: "Domain deleted",
      error: (err) => ({
        message: "Failed to delete domain",
        description: err.message,
      }),
    });

    try {
      await mutation;
      onDelete();
    } catch {}
  };

  const handleRetry = async () => {
    const mutation = retryMutation.mutateAsync({
      domain: domain.domain,
      projectId,
    });

    toast.promise(mutation, {
      loading: "Retrying verification...",
      success: "Verification restarted",
      error: (err) => ({
        message: "Failed to retry verification",
        description: err.message,
      }),
    });

    try {
      await mutation;
      onRetry();
    } catch {}
  };

  const isLoading = deleteMutation.isLoading || retryMutation.isLoading;

  return (
    <div className="border-b border-gray-4 last:border-b-0 group hover:bg-gray-2 transition-colors">
      <div className="flex items-center justify-between px-4 py-3">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <Link4 className="text-gray-9 !size-4 flex-shrink-0" />
          <a
            href={`https://${domain.domain}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-content font-medium hover:underline truncate"
          >
            {domain.domain}
          </a>
        </div>

        <div className="flex items-center gap-3">
          <Badge variant={status.color} className="gap-1">
            {status.icon}
            {status.label}
          </Badge>

          {verificationStatus === "failed" && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  onClick={handleRetry}
                  disabled={isLoading}
                  className="size-7 text-gray-9 hover:text-gray-11"
                >
                  <Refresh3
                    className={cn("!size-3.5", retryMutation.isLoading && "animate-spin")}
                  />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Retry verification</TooltipContent>
            </Tooltip>
          )}

          {domain.verificationError && (
            <Tooltip>
              <TooltipTrigger>
                <CircleInfo className="!size-4 text-error-9" />
              </TooltipTrigger>
              <TooltipContent className="max-w-xs">{domain.verificationError}</TooltipContent>
            </Tooltip>
          )}

          <Button
            ref={deleteButtonRef}
            size="icon"
            variant="ghost"
            disabled={isLoading}
            onClick={() => setIsConfirmOpen(true)}
            className="size-7 text-gray-9 hover:text-error-9 opacity-0 group-hover:opacity-100 transition-opacity"
          >
            <Trash className="!size-3.5" />
          </Button>

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

      {verificationStatus !== "verified" && (
        <DnsRecordTable
          domain={domain.domain}
          targetCname={domain.targetCname}
          verificationToken={domain.verificationToken}
          ownershipVerified={domain.ownershipVerified}
          cnameVerified={domain.cnameVerified}
          projectId={projectId}
        />
      )}
    </div>
  );
}

type DnsRecordTableProps = {
  domain: string;
  targetCname: string;
  verificationToken: string;
  ownershipVerified: boolean;
  cnameVerified: boolean;
  projectId: string;
};

// Backend checks every 60 seconds via Restate
const CHECK_INTERVAL_MS = 60 * 1000;

function DnsRecordTable({
  domain,
  targetCname,
  verificationToken: initialVerificationToken,
  ownershipVerified: initialOwnershipVerified,
  cnameVerified: initialCnameVerified,
  projectId,
}: DnsRecordTableProps) {
  const [secondsUntilCheck, setSecondsUntilCheck] = useState<number>(CHECK_INTERVAL_MS / 1000);

  // Poll for DNS status updates - only fetches this specific domain
  const { data: dnsStatus, dataUpdatedAt } = trpc.deploy.customDomain.checkDns.useQuery(
    { domain, projectId },
    {
      refetchInterval: CHECK_INTERVAL_MS,
      refetchIntervalInBackground: false,
    },
  );

  // Use live data if available, otherwise fall back to initial props
  const verificationToken = dnsStatus?.verificationToken ?? initialVerificationToken;
  const ownershipVerified = dnsStatus?.ownershipVerified ?? initialOwnershipVerified;
  const cnameVerified = dnsStatus?.cnameVerified ?? initialCnameVerified;

  useEffect(() => {
    const calculateSecondsRemaining = () => {
      if (!dataUpdatedAt) {
        return CHECK_INTERVAL_MS / 1000;
      }
      const nextCheckAt = dataUpdatedAt + CHECK_INTERVAL_MS;
      const remaining = Math.max(0, Math.ceil((nextCheckAt - Date.now()) / 1000));
      return remaining;
    };

    setSecondsUntilCheck(calculateSecondsRemaining());

    const interval = setInterval(() => {
      setSecondsUntilCheck(calculateSecondsRemaining());
    }, 1000);

    return () => clearInterval(interval);
  }, [dataUpdatedAt]);

  const txtRecordName = `_unkey.${domain}`;
  const txtRecordValue = `unkey-domain-verify=${verificationToken}`;

  return (
    <div className="px-4 pb-3 space-y-4">
      <p className="text-xs text-gray-9">Add both DNS records below at your domain provider.</p>

      {/* TXT Record (Ownership Verification) */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <p className="text-xs text-gray-11 font-medium">TXT Record (ownership)</p>
          <StatusIndicator
            verified={ownershipVerified}
            label={ownershipVerified ? "Verified" : "Pending"}
          />
        </div>
        <div className="border border-gray-4 rounded-lg overflow-hidden text-xs">
          <div className="grid grid-cols-[80px_1fr_1fr_60px] bg-gray-3 px-3 py-1.5 text-gray-9 font-medium">
            <span>Type</span>
            <span>Name</span>
            <span>Value</span>
            <span>Status</span>
          </div>
          <div className="grid grid-cols-[80px_1fr_1fr_60px] px-3 py-2 items-center">
            <span className="text-gray-11 font-medium">TXT</span>
            <span className="flex items-center gap-1.5 min-w-0">
              <code className="text-content font-mono truncate">{txtRecordName}</code>
              <CopyButton value={txtRecordName} variant="ghost" className="size-5 flex-shrink-0" />
            </span>
            <span className="flex items-center gap-1.5 min-w-0">
              <code className="text-content font-mono truncate">{txtRecordValue}</code>
              <CopyButton value={txtRecordValue} variant="ghost" className="size-5 flex-shrink-0" />
            </span>
            <span className="flex justify-center">
              {ownershipVerified ? (
                <CircleCheck className="!size-4 text-success-9" />
              ) : (
                <XMark className="!size-4 text-gray-7" />
              )}
            </span>
          </div>
        </div>
      </div>

      {/* CNAME Record (Routing) */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <p className="text-xs text-gray-11 font-medium">CNAME Record (routing)</p>
          <StatusIndicator
            verified={cnameVerified}
            label={cnameVerified ? "Verified" : "Pending"}
          />
        </div>
        <div className="border border-gray-4 rounded-lg overflow-hidden text-xs">
          <div className="grid grid-cols-[80px_1fr_1fr_60px] bg-gray-3 px-3 py-1.5 text-gray-9 font-medium">
            <span>Type</span>
            <span>Name</span>
            <span>Value</span>
            <span>Status</span>
          </div>
          <div className="grid grid-cols-[80px_1fr_1fr_60px] px-3 py-2 items-center">
            <span className="text-gray-11 font-medium">CNAME</span>
            <span className="flex items-center gap-1.5 min-w-0">
              <code className="text-content font-mono truncate">{domain}</code>
              <CopyButton value={domain} variant="ghost" className="size-5 flex-shrink-0" />
            </span>
            <span className="flex items-center gap-1.5 min-w-0">
              <code className="text-content font-mono truncate">{targetCname}</code>
              <CopyButton value={targetCname} variant="ghost" className="size-5 flex-shrink-0" />
            </span>
            <span className="flex justify-center">
              {cnameVerified ? (
                <CircleCheck className="!size-4 text-success-9" />
              ) : (
                <XMark className="!size-4 text-gray-7" />
              )}
            </span>
          </div>
        </div>
      </div>

      {/* Next check countdown */}
      <div className="flex justify-end">
        <span className="text-xs text-gray-9 flex items-center gap-1.5">
          <Refresh3 className="!size-3.5" />
          {secondsUntilCheck <= 1 ? "Refreshing..." : `Next check in ${secondsUntilCheck}s`}
        </span>
      </div>
    </div>
  );
}

function StatusIndicator({ verified, label }: { verified: boolean; label: string }) {
  return (
    <Badge variant={verified ? "success" : "secondary"} className="gap-1 text-xs">
      {verified ? <CircleCheck className="!size-3" /> : <Clock className="!size-3" />}
      {label}
    </Badge>
  );
}

export function CustomDomainRowSkeleton() {
  return (
    <div className="flex items-center justify-between px-4 py-3 border-b border-gray-4 last:border-b-0">
      <div className="flex items-center gap-3">
        <div className="w-4 h-4 bg-gray-4 rounded animate-pulse" />
        <div className="w-32 h-4 bg-gray-4 rounded animate-pulse" />
      </div>
      <div className="w-16 h-5 bg-gray-4 rounded animate-pulse" />
    </div>
  );
}
