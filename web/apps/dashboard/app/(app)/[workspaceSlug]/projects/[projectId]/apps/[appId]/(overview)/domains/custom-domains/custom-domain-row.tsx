"use client";
import { collection } from "@/lib/collections";
import {
  type CustomDomain,
  type VerificationStatus,
  retryDomainVerification,
} from "@/lib/collections/deploy/custom-domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import {
  IconChevronDownOutline12,
  IconCircleCheckOutline18,
  IconClockOutline18,
  IconRefresh3Outline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import {
  Alert,
  AlertDescription,
  Button,
  ConfirmPopover,
  Loading,
  ResourceListRow,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@unkey/ui";
import { cn } from "cn";
import { useRef, useState } from "react";
import { EnvironmentLabel } from "../../../components/environment-label";
import { RemoveButton } from "../../settings/components/shared/remove-button";
import { DnsRecordTable } from "./dns-record-table";

export const CUSTOM_DOMAIN_COLUMNS =
  "grid grid-cols-[24px_minmax(0,1.6fr)_minmax(0,1fr)_minmax(0,0.7fr)_64px_12px] items-center gap-4";

type CustomDomainRowProps = {
  domain: CustomDomain;
  environment?: Environment;
  defaultExpanded: boolean;
};

const statusConfig: Record<
  VerificationStatus,
  { label: string; labelClassName: string; icon: React.ReactNode }
> = {
  pending: {
    label: "Waiting for DNS records",
    labelClassName: "text-gray-11",
    icon: <IconClockOutline18 className="size-4 text-gray-9" />,
  },
  verifying: {
    label: "Verifying",
    labelClassName: "text-gray-11",
    icon: <Loading size={16} className="text-gray-9" />,
  },
  verified: {
    label: "Verified",
    labelClassName: "text-gray-11",
    icon: <IconCircleCheckOutline18 className="size-4 text-success-11" />,
  },
  failed: {
    label: "Verification failed",
    labelClassName: "text-error-11",
    icon: <IconTriangleWarningOutline18 className="size-4 text-error-11" />,
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

export function CustomDomainRow({ domain, environment, defaultExpanded }: CustomDomainRowProps) {
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [isRetrying, setIsRetrying] = useState(false);
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);
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
    <div>
      <ResourceListRow
        label={`Details for ${domain.domain}`}
        expanded={isExpanded}
        active={isExpanded}
        onActivate={() => setIsExpanded((open) => !open)}
        className={cn(CUSTOM_DOMAIN_COLUMNS, "text-xs")}
      >
        <span className="flex size-6 items-center justify-center">{status.icon}</span>
        <span className="flex min-w-0">
          <a
            href={`https://${domain.domain}`}
            target="_blank"
            rel="noopener noreferrer"
            className="relative truncate text-sm font-medium text-gray-12 hover:underline"
          >
            {domain.domain}
          </a>
        </span>
        <span className={cn("truncate", status.labelClassName)}>{status.label}</span>
        <span className="flex min-w-0 items-center">
          <EnvironmentLabel environment={environment} className="text-xs" />
        </span>
        <span className="relative flex items-center justify-end gap-1">
          {domain.verificationStatus === "failed" && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    size="icon"
                    variant="ghost"
                    aria-label="Retry verification"
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
          <RemoveButton
            onClick={() => setIsConfirmOpen(true)}
            ref={deleteButtonRef}
            className="size-7 text-gray-9 hover:text-gray-12"
          />
          {deleteButtonRef.current && (
            <ConfirmPopover
              isOpen={isConfirmOpen}
              onOpenChange={setIsConfirmOpen}
              triggerRef={deleteButtonRef}
              title="Delete domain"
              description={`${domain.domain} stops routing to this app.`}
              onConfirm={handleDelete}
              confirmButtonText="Delete domain"
              variant="danger"
            />
          )}
        </span>
        <IconChevronDownOutline12
          aria-hidden
          className={cn("text-gray-9 transition-transform", isExpanded && "rotate-180")}
        />
      </ResourceListRow>

      {isExpanded ? (
        <div className="grid animate-expand-down overflow-hidden">
          <div className="min-h-0">
            <CustomDomainDetails domain={domain} />
          </div>
        </div>
      ) : null}
    </div>
  );
}

function CustomDomainDetails({ domain }: { domain: CustomDomain }) {
  const { verificationError, domainConnectProvider, domainConnectUrl } = domain;
  const isVerified = domain.verificationStatus === "verified";

  return (
    <div className="flex flex-col gap-3 border-t bg-raised px-4 py-4">
      {verificationError && (
        <Alert variant="alert" role="note">
          <AlertDescription className="text-xs">{verificationError}</AlertDescription>
        </Alert>
      )}
      {!isVerified && domainConnectProvider && domainConnectUrl && (
        <div className="flex items-center gap-3 rounded-lg border bg-background px-4 py-3">
          <ProviderIcon provider={domainConnectProvider} className="size-6!" />
          <div className="flex-1">
            <p className="text-sm font-medium text-gray-12">Automatic setup available</p>
            <p className="text-xs text-gray-9">
              We detected your domain uses {domainConnectProvider}. We can configure your DNS
              records automatically.
            </p>
          </div>
          <Button
            variant="primary"
            onClick={() => window.open(domainConnectUrl, "_blank", "noopener,noreferrer")}
          >
            Connect
          </Button>
        </div>
      )}
      <DnsRecordTable records={domain.dnsRecords} verified={isVerified} />
    </div>
  );
}
