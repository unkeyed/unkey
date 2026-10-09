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
import { match } from "@unkey/match";
import {
  Alert,
  AlertDescription,
  Button,
  ConfirmPopover,
  Loading,
  ResourceListItem,
  ResourceListRow,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@unkey/ui";
import { cn } from "cn";
import { useRef, useState } from "react";
import { EnvironmentLabel } from "../../../../_components/environment-label";
import { RemoveButton } from "../../../../_components/remove-button";
import type { DisplayDomain, DisplayDomainCustom } from "../../../components/display-domain";
import { DnsRecordTable } from "./dns-record-table";
import { ProviderIcon } from "./provider-icon";

type RowStatus = VerificationStatus | "platform";

const statusConfig: Record<
  RowStatus,
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
  platform: {
    label: "Active",
    labelClassName: "text-gray-11",
    icon: <IconCircleCheckOutline18 className="size-4 text-success-11" />,
  },
};

type DomainColumnsProps = React.ComponentProps<"div"> & {
  activate?: { label: string; onActivate: () => void; expanded: boolean };
};

export function DomainColumns({ activate, className, ...props }: DomainColumnsProps) {
  const columns = cn(
    "grid h-12 grid-cols-(--domain-columns) items-center gap-4 px-4 text-xs",
    className,
  );
  return activate ? (
    <ResourceListRow {...activate} className={columns} {...props} />
  ) : (
    <div className={columns} {...props} />
  );
}

type DomainRowProps = {
  domain: DisplayDomain;
  environment?: Environment;
  expanded: boolean;
  onToggle: () => void;
};

export function DomainRow({ domain, environment, expanded, onToggle }: DomainRowProps) {
  return match(domain)
    .with({ source: "custom" }, (display) => (
      <CustomDomainRow
        display={display}
        environment={environment}
        expanded={expanded}
        onToggle={onToggle}
      />
    ))
    .with({ source: "platform" }, ({ hostname, url }) => (
      <ResourceListItem>
        <DomainColumns>
          <DomainCells hostname={hostname} url={url} status="platform" environment={environment} />
        </DomainColumns>
      </ResourceListItem>
    ))
    .exhaustive();
}

function DomainCells({
  hostname,
  url,
  status,
  environment,
}: {
  hostname: string;
  url: string;
  status: RowStatus;
  environment?: Environment;
}) {
  const { icon, label, labelClassName } = statusConfig[status];
  return (
    <>
      <span className="flex size-6 items-center justify-center">{icon}</span>
      <span className="flex min-w-0">
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className="relative truncate text-sm font-medium text-gray-12 hover:underline"
        >
          {hostname}
        </a>
      </span>
      <span className={cn("truncate", labelClassName)}>{label}</span>
      <span className="flex min-w-0 items-center">
        <EnvironmentLabel environment={environment} />
      </span>
    </>
  );
}

function CustomDomainRow({
  display,
  environment,
  expanded,
  onToggle,
}: {
  display: DisplayDomainCustom;
  environment?: Environment;
  expanded: boolean;
  onToggle: () => void;
}) {
  const { customDomain: domain, hostname } = display;
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [isRetrying, setIsRetrying] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const handleDelete = () => {
    collection.customDomains.delete(domain.id);
  };

  const handleRetry = async () => {
    setIsRetrying(true);
    try {
      await retryDomainVerification({ domain: hostname });
    } finally {
      setIsRetrying(false);
    }
  };

  return (
    <ResourceListItem>
      <DomainColumns
        activate={{ label: `Details for ${hostname}`, onActivate: onToggle, expanded }}
      >
        <DomainCells
          hostname={hostname}
          url={display.url}
          status={domain.verificationStatus}
          environment={environment}
        />
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
            label={`Delete ${hostname}`}
            onClick={() => setIsConfirmOpen(true)}
            ref={deleteButtonRef}
            className="size-7 text-gray-9 hover:text-gray-12"
          />
          <ConfirmPopover
            isOpen={isConfirmOpen}
            onOpenChange={setIsConfirmOpen}
            triggerRef={deleteButtonRef}
            title="Delete domain"
            description={`${hostname} stops routing to this app.`}
            onConfirm={handleDelete}
            confirmButtonText="Delete domain"
            variant="danger"
          />
        </span>
        <IconChevronDownOutline12
          aria-hidden
          className={cn("text-gray-9 transition-transform", expanded && "rotate-180")}
        />
      </DomainColumns>

      {expanded ? (
        <div className="grid animate-expand-down overflow-hidden">
          <div className="min-h-0">
            <CustomDomainDetails domain={domain} />
          </div>
        </div>
      ) : null}
    </ResourceListItem>
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
