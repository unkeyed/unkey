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

function CloudflareIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 65 32" fill="currentColor">
      <path d="M45.234 24.397l.517-1.808c.345-1.193.19-2.29-.434-3.093-.58-.748-1.503-1.173-2.6-1.228l-18.238-.248a.37.37 0 01-.31-.18.4.4 0 01-.034-.37c.069-.165.228-.275.407-.283l18.393-.248c2.697-.138 5.624-2.345 6.634-5.017l1.276-3.38a.63.63 0 00.034-.275C49.019 3.938 44.826 0 39.725 0c-4.47 0-8.277 2.842-9.722 6.82-.89-.675-2.014-1.076-3.228-1.02-2.207.103-3.978 1.89-4.296 4.098-.076.51-.07 1.007.014 1.476C17.907 11.553 14 15.614 14 20.573c0 .648.062 1.283.165 1.904a.62.62 0 00.607.524l29.82.007c.2-.014.386-.138.462-.324l.18-.29z" />
      <path d="M49.124 11.374a.49.49 0 00-.476.048.479.479 0 00-.207.4l-.276 1.724c-.345 1.193-.19 2.29.434 3.093.58.748 1.503 1.173 2.6 1.228l3.89.248c.172.014.324.103.4.234a.4.4 0 01.034.37c-.069.165-.228.275-.407.283l-4.048.248c-2.704.138-5.631 2.345-6.641 5.017l-.358.952a.26.26 0 00.234.358h13.107a.55.55 0 00.524-.386A11.425 11.425 0 0060 20.573c0-5.117-3.345-9.447-7.959-10.924a5.506 5.506 0 00-2.917 1.724z" />
    </svg>
  );
}

function VercelIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 74 64" fill="currentColor" aria-label="Vercel logomark">
      <path d="M37.5896 0.25L74.5396 64.25H0.639648L37.5896 0.25Z" />
    </svg>
  );
}

const providerIcons: Record<string, (props: { className?: string }) => React.ReactNode> = {
  cloudflare: CloudflareIcon,
  vercel: VercelIcon,
};

function ProviderIcon({ provider, className }: { provider: string; className?: string }) {
  const key = provider.toLowerCase().replace(/[.\s]+/g, "");
  const Icon = providerIcons[key];
  return Icon ? <Icon className={className} /> : null;
}

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

      {domain.verificationStatus !== "verified" &&
        domain.domainConnectUrl &&
        domain.domainConnectProvider && (
          <div className="mx-4 mb-3 flex items-center gap-3 px-4 py-3 rounded-lg border border-gray-4 bg-gray-2">
            <ProviderIcon provider={domain.domainConnectProvider} className="size-6!" />
            <div className="flex-1">
              <p className="text-[13px] font-medium text-gray-12">Automatic setup available</p>
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
        <DnsRecordTable
          domain={domain.domain}
          targetCname={domain.targetCname}
          verificationToken={domain.verificationToken}
          ownershipVerified={domain.ownershipVerified}
          cnameVerified={domain.cnameVerified}
          isLoading={!domain.targetCname}
        />
      )}
    </div>
  );
}
