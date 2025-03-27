"use client";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import BackButton from "@/components/ui/back-button";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { formatNumber } from "@/lib/fmt";
import { Key } from "@unkey/icons";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { ChevronRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

type KeyData = {
  id: string;
  keyAuthId: string;
  name: string | null;
  start: string | null;
  roles: number;
  permissions: number;
  enabled: boolean;
  environment: string | null;
  externalId: string | null;
};

type Props = {
  keys: KeyData[];
  apiId: string;
  keyAuthId: string;
};

export const VirtualKeys: React.FC<Props> = ({ keys, apiId, keyAuthId }) => {
  const router = useRouter();
  const [selectedKey, setSelectedKey] = useState<KeyData | null>(null);

  const handleRowClick = (key: KeyData) => {
    setSelectedKey(key);
    router.push(`/apis/${apiId}/keys/${key.keyAuthId}/${key.id}`);
  };

  console.log({ keys });

  const columns = (): Column<KeyData>[] => {
    return [
      {
        key: "key",
        header: "Key",
        width: "25%",
        headerClassName: "pl-[18px]",
        render: (key) => (
          <div className="flex flex-col items-start px-[18px] py-[6px]">
            <div className="flex gap-4 items-center">
              <div className="bg-grayA-3 size-5 rounded flex items-center justify-center">
                <Key size="sm-regular" />
              </div>
              <div className="flex flex-col gap-1 text-xs">
                <div className="font-mono font-medium truncate text-brand-12">
                  {key.id.substring(0, 8)}...
                  {key.id.substring(key.id.length - 4)}
                </div>

                <span className="font-sans text-accent-9">{key.name}</span>
              </div>
            </div>
          </div>
        ),
      },
      {
        key: "value",
        header: "Value",
        width: "20%",
        render: (key) => <div className="flex items-center">{key.permissions}</div>,
      },
      {
        key: "environment",
        header: "Environment",
        width: "10%",
        render: (key) => (
          <div className="flex items-center gap-2">
            {key.environment ? <Badge variant="secondary">env: {key.environment}</Badge> : null}
          </div>
        ),
      },
      {
        key: "permissions",
        header: "Permissions",
        width: "15%",
        render: (key) => (
          <Badge variant="secondary">
            {formatNumber(key.permissions)} Permission
            {key.permissions !== 1 ? "s" : ""}
          </Badge>
        ),
      },
      {
        key: "roles",
        header: "Roles",
        width: "15%",
        render: (key) => (
          <Badge variant="secondary">
            {formatNumber(key.roles)} Role{key.roles !== 1 ? "s" : ""}
          </Badge>
        ),
      },
      {
        key: "status",
        header: "Status",
        width: "10%",
        render: (key) => <div>{!key.enabled && <Badge variant="secondary">Disabled</Badge>}</div>,
      },
      {
        key: "actions",
        header: "",
        width: "5%",
        render: () => (
          <div className="flex items-center justify-end">
            <Button variant="ghost">
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        ),
      },
    ];
  };

  // If there are no keys, show the empty state
  if (keys.length === 0) {
    return (
      <Empty>
        <Empty.Icon />
        <Empty.Title>No keys found</Empty.Title>
        <Empty.Description>Create your first key</Empty.Description>
        <Empty.Actions>
          <CreateKeyButton apiId={apiId} keyAuthId={keyAuthId} />
          <BackButton className="ml-4">Go Back</BackButton>
        </Empty.Actions>
      </Empty>
    );
  }

  return (
    <div className="flex flex-col gap-8 mb-20">
      <VirtualTable
        data={keys}
        columns={columns()}
        onRowClick={handleRowClick}
        selectedItem={selectedKey}
        keyExtractor={(key) => key.id}
        config={{
          rowHeight: 52,
          layoutMode: "grid",
          rowBorders: true,
          containerPadding: "px-0",
        }}
      />
    </div>
  );
};
