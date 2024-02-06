"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Loading } from "@/components/dashboard/loading";
import { VisibleButton } from "@/components/dashboard/visible-button";
import { Button } from "@/components/ui/button";
import { Code } from "@/components/ui/code";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { trpc } from "@/lib/trpc/client";
import { UnkeyPermission } from "@unkey/rbac";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { apiPermissions, workspacePermissions } from "../[keyId]/permissions/permissions";

type Props = {
  apis: {
    id: string;
    name: string;
  }[];
};

export const Client: React.FC<Props> = ({ apis }) => {
  const router = useRouter();
  const [name, setName] = useState<string | undefined>(undefined);
  const [selectedPermissions, setSelectedPermissions] = useState<UnkeyPermission[]>([]);

  const key = trpc.key.createInternalRootKey.useMutation({
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const snippet = `curl -XPOST '${process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"}/v1/keys.createKey' \\
  -H 'Authorization: Bearer ${key.data?.key}' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "prefix": "hello",
    "apiId": "<API_ID>"
  }'`;

  const maskedKey = `unkey_${"*".repeat(key.data?.key.split("_").at(1)?.length ?? 0)}`;
  const [showKey, setShowKey] = useState(false);
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);

  const handleSetChecked = (permission: UnkeyPermission, checked: boolean) => {
    setSelectedPermissions((prevPermissions) => {
      if (checked) {
        return [...prevPermissions, permission];
      } else {
        return prevPermissions.filter((r) => r !== permission);
      }
    });
  };

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Name</CardTitle>
          <CardDescription>
            Give your key a name. This is optional and not customer facing.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Vercel Production"
          />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Workspace</CardTitle>
          <CardDescription>Manage workspace permissions</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-4">
            {Object.entries(workspacePermissions).map(([category, allPermissions]) => (
              <div className="flex flex-col gap-2">
                <span className="font-medium">{category}</span>{" "}
                <div className="flex flex-col gap-1">
                  {Object.entries(allPermissions).map(([action, { description, permission }]) => (
                    <PermissionToggle
                      key={action}
                      permissionName={permission}
                      label={action}
                      description={description}
                      checked={selectedPermissions.includes(permission)}
                      setChecked={(c) => handleSetChecked(permission, c)}
                    />
                  ))}
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
      {apis.map((api) => (
        <Card>
          <CardHeader>
            <CardTitle>{api.name}</CardTitle>
            <CardDescription>
              Permissions scoped to this API. Enabling these roles only grants access to this
              specific API.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-4">
              {Object.entries(apiPermissions(api.id)).map(([category, roles]) => {
                return (
                  <div className="flex flex-col gap-2">
                    <span className="font-medium">{category}</span>
                    <div className="flex flex-col gap-1">
                      {Object.entries(roles).map(([action, { description, permission }]) => {
                        return (
                          <PermissionToggle
                            key={action}
                            permissionName={permission}
                            label={action}
                            description={description}
                            checked={selectedPermissions.includes(permission)}
                            setChecked={(c) => handleSetChecked(permission, c)}
                          />
                        );
                      })}
                    </div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      ))}

      <Button
        onClick={() => {
          key.mutate({
            name: name && name.length > 0 ? name : undefined,
            permissions: selectedPermissions,
          });
        }}
      >
        {key.isLoading ? <Loading className="w-4 h-4" /> : "Create New Key"}
      </Button>

      <Dialog
        open={!!key.data?.key}
        onOpenChange={(v) => {
          if (!v) {
            // Remove the key from memory when closing the modal
            key.reset();
            setSelectedPermissions([]);
            router.refresh();
          }
        }}
      >
        <DialogContent className="flex flex-col max-sm:w-full">
          <DialogHeader>
            <DialogTitle>Your API Key</DialogTitle>
            <DialogDescription className="w-fit">
              This key is only shown once and can not be recovered. Please store it somewhere safe.
            </DialogDescription>

            <Code className="flex items-center justify-between gap-4 my-8 ph-no-capture">
              {showKey ? key.data?.key : maskedKey}
              <div className="flex items-start justify-between gap-4 ">
                <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
                <CopyButton value={key.data?.key ?? ""} />
              </div>
            </Code>
          </DialogHeader>

          <p className="mt-2 text-sm font-medium text-center text-gray-700 ">
            Try creating a new api key for your users:
          </p>
          <Code className="flex items-start justify-between gap-4 pt-10 my-8 text-xs ">
            {showKeyInSnippet ? snippet : snippet.replace(key.data?.key ?? "", maskedKey)}
            <div className="relative -top-8 right-[88px] flex items-start justify-between gap-4">
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
              <CopyButton value={snippet} />
            </div>
          </Code>
          <DialogClose asChild>
            <Button type="button" variant="primary">
              I have copied the key and want to close this dialog
            </Button>
          </DialogClose>
        </DialogContent>
      </Dialog>
    </div>
  );
};

type PermissionToggleProps = {
  checked: boolean;
  setChecked: (checked: boolean) => void;
  permissionName: string;
  label: string;
  description: string;
};

const PermissionToggle: React.FC<PermissionToggleProps> = ({
  checked,
  setChecked,
  permissionName,
  label,
  description,
}) => {
  return (
    <div className="flex items-center gap-8">
      <div className="w-1/3 ">
        <Tooltip>
          <TooltipTrigger className="flex items-center gap-2">
            <Checkbox
              checked={checked}
              onClick={() => {
                setChecked(!checked);
              }}
            />

            <Label className="text-xs text-content">{label}</Label>
          </TooltipTrigger>
          <TooltipContent className="flex items-center gap-2">
            <span className="font-mono text-sm font-medium">{permissionName}</span>
            <CopyButton value={permissionName} />
          </TooltipContent>
        </Tooltip>
      </div>

      <p className="w-2/3 text-xs text-content-subtle">{description}</p>
    </div>
  );
};
