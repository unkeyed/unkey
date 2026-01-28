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
} from "@unkey/icons";
import {
  Badge,
  Button,
  ConfirmPopover,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  toast,
} from "@unkey/ui";
import { useMemo, useRef, useState } from "react";
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
  const anchorRef = useMemo(
    () => ({ current: deleteButtonRef.current }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [deleteButtonRef.current],
  );

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
    <div className="flex items-center justify-between px-4 py-3 border-b border-gray-4 last:border-b-0 group hover:bg-gray-2 transition-colors">
      <div className="flex items-center gap-3 flex-1 min-w-0">
        <Link4 className="text-gray-9 !size-4 flex-shrink-0" />
        <div className="flex flex-col min-w-0">
          <a
            href={`https://${domain.domain}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-content font-medium hover:underline truncate"
          >
            {domain.domain}
          </a>
          {verificationStatus !== "verified" && (
            <CnameInstructions targetCname={domain.targetCname} domain={domain.domain} />
          )}
        </div>
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
                <Refresh3 className={cn("!size-3.5", retryMutation.isLoading && "animate-spin")} />
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
            triggerRef={anchorRef}
            title="Delete domain"
            description={`Are you sure you want to delete ${domain.domain}? This will remove the domain and any associated routing.`}
            onConfirm={handleDelete}
            confirmButtonText="Delete"
            variant="danger"
          />
        )}
      </div>
    </div>
  );
}

function CnameInstructions({ targetCname, domain }: { targetCname: string; domain: string }) {
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  };

  return (
    <div className="flex items-center gap-1 text-xs text-gray-9 mt-0.5">
      <span>Add CNAME:</span>
      <button
        type="button"
        onClick={() => copyToClipboard(domain)}
        className="font-mono text-accent-11 hover:underline cursor-pointer"
      >
        {domain}
      </button>
      <span>→</span>
      <button
        type="button"
        onClick={() => copyToClipboard(targetCname)}
        className="font-mono text-accent-11 hover:underline cursor-pointer"
      >
        {targetCname}
      </button>
    </div>
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
