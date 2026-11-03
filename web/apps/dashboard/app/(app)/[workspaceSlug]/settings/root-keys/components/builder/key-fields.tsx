"use client";

import { ROOT_KEY_IP_ALLOWLIST_DOCS, SUPPORT_MAILTO } from "@/lib/support";
import { IconCircleInfoOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Badge,
  type BadgeProps,
  Button,
  FormInput,
  FormTextarea,
  InfoTooltip,
  Item,
  ItemContent,
  ItemDescription,
  ItemTitle,
  RequiredTag,
  VisuallyHidden,
} from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";
import { Controller, useFormContext, useFormState } from "react-hook-form";
import type { IpAllowlistMode, RootKeyFormValues } from "./schema";

type NamedFormValues = {
  name: string;
};

function NameField() {
  const { control } = useFormContext<NamedFormValues>();

  return (
    <Controller
      control={control}
      name="name"
      render={({ field, fieldState }) => (
        <FormInput
          label="Name"
          requirement="required"
          placeholder="e.g. CI deploy key"
          ref={field.ref}
          value={field.value}
          onChange={field.onChange}
          error={fieldState.error?.message}
        />
      )}
    />
  );
}

const IP_ALLOWLIST_LABEL = "IP allowlist";
const IP_ALLOWLIST_TOOLTIP = "This restricts callers that use this root key, not your end users.";

type IpAllowlistNoticeMode = Exclude<IpAllowlistMode, { kind: "editable" }>;

function EditableIpAllowlist() {
  const { control } = useFormContext<RootKeyFormValues>();

  return (
    <Controller
      control={control}
      name="ipAllowlist"
      render={({ field, fieldState }) => (
        <FormTextarea
          {...field}
          label={IP_ALLOWLIST_LABEL}
          rows={4}
          placeholder={"203.0.113.10\n198.51.100.0/24\n2001:db8::/32"}
          description="Enter one IP address or CIDR range per line. Leave empty to allow any IP address."
          tooltip={IP_ALLOWLIST_TOOLTIP}
          error={fieldState.error?.message}
        />
      )}
    />
  );
}

const IP_ALLOWLIST_NOTICES: Record<
  IpAllowlistNoticeMode["kind"],
  { badge: { label: string; variant: BadgeProps["variant"] }; description: string }
> = {
  locked: {
    badge: { label: "Add-on", variant: "primary" },
    description: "Restrict this root key to the IP addresses and CIDR ranges you trust.",
  },
  enforced: {
    badge: { label: "Enforced", variant: "success" },
    description: "Only these addresses can use this root key. Upgrade to edit them.",
  },
};

function IpAllowlistNotice({
  kind,
  children,
}: {
  kind: IpAllowlistNoticeMode["kind"];
  children?: ReactNode;
}) {
  const { badge, description } = IP_ALLOWLIST_NOTICES[kind];

  return (
    <Item className="items-start p-0">
      <ItemContent className="gap-1">
        <ItemTitle className="flex items-center gap-2">
          {IP_ALLOWLIST_LABEL}
          <Badge variant={badge.variant} size="compact">
            {badge.label}
          </Badge>
        </ItemTitle>
        <ItemDescription className="text-sm leading-5">{description}</ItemDescription>
        {children}
        <div className="mt-2 flex items-center gap-2">
          <Button variant="outline" size="sm" render={<Link href={SUPPORT_MAILTO} />}>
            Contact support to upgrade
          </Button>
          <Button
            variant="ghost"
            size="sm"
            render={
              <Link href={ROOT_KEY_IP_ALLOWLIST_DOCS} target="_blank" rel="noopener noreferrer" />
            }
          >
            Learn more
            <VisuallyHidden> (opens in new tab)</VisuallyHidden>
          </Button>
        </div>
      </ItemContent>
    </Item>
  );
}

function IpAllowlistField({ mode }: { mode: IpAllowlistMode }) {
  return match(mode)
    .with({ kind: "editable" }, () => <EditableIpAllowlist />)
    .with({ kind: "locked" }, () => <IpAllowlistNotice kind="locked" />)
    .with({ kind: "enforced" }, ({ entries }) => (
      <IpAllowlistNotice kind="enforced">
        <ul className="mt-1 flex flex-wrap gap-1.5">
          {entries.map((entry, index) => (
            <li key={`${index}-${entry}`} className="flex">
              <Badge variant="code">{entry}</Badge>
            </li>
          ))}
        </ul>
      </IpAllowlistNotice>
    ))
    .exhaustive();
}

type KeyFieldsProps = {
  children: ReactNode;
  nameHidden?: boolean;
  ipAllowlist?: IpAllowlistMode;
};

export function KeyFields({ children, nameHidden = false, ipAllowlist }: KeyFieldsProps) {
  const { control } = useFormContext<RootKeyFormValues>();
  const { errors } = useFormState({ control, name: "policies" });

  return (
    <div className="flex flex-col gap-6">
      {nameHidden ? null : <NameField />}

      {ipAllowlist === undefined ? null : <IpAllowlistField mode={ipAllowlist} />}

      <div className="flex flex-col gap-2">
        <span className="flex h-5 items-center text-sm text-gray-11">
          Permissions
          <InfoTooltip
            content="Select the privileges you'd like this Root Key to have."
            position={{ side: "right" }}
          >
            <IconCircleInfoOutline18 className="ml-1.5 size-3.5 shrink-0 text-gray-9" />
          </InfoTooltip>
          <RequiredTag hasError={errors.policies !== undefined} />
        </span>
        {children}
      </div>
    </div>
  );
}
