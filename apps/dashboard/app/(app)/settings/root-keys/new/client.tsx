"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Loading } from "@/components/dashboard/loading";
import { VisibleButton } from "@/components/dashboard/visible-button";
import { Button } from "@/components/ui/button";
import { Code } from "@/components/ui/code";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
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
import type { UnkeyPermission } from "@unkey/rbac";
import { ChevronRight } from "lucide-react";
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

  const key = trpc.rootKey.create.useMutation({
    onError(err: { message: string }) {
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
      }
      return prevPermissions.filter((r) => r !== permission);
    });
  };

  type CardStates = {
    [key: string]: boolean;
  };

  const initialCardStates: CardStates = {};
  apis.forEach((api) => {
    initialCardStates[api.id] = false;
  });
  const [cardStatesMap, setCardStatesMap] = useState(initialCardStates);

  const toggleCard = (apiId: string) => {
    setCardStatesMap((prevStates) => ({
      ...prevStates,
      [apiId]: !prevStates[apiId],
    }));
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
            {Object.entries(workspacePermissions).map(([category, allPermissions]) => {
              const allPermissionNames = Object.values(allPermissions).map(
                ({ permission }) => permission,
              );
              const isAllSelected = allPermissionNames.every((permission) =>
                selectedPermissions.includes(permission),
              );

              return (
                <div key={`workspace-${category}`} className="flex flex-col gap-2">
                  <div className="flex flex-col">
                    <PermissionToggle
                      permissionName={`selectAll-${category}`}
                      label={<span className="text-base font-bold">{category}</span>}
                      description={`Select all permissions for ${category} in this workspace`}
                      checked={isAllSelected}
                      setChecked={(isChecked) => {
                        allPermissionNames.forEach((permission) => {
                          handleSetChecked(permission, isChecked);
                        });
                      }}
                    />
                  </div>

                  <div className="flex flex-col gap-1">
                    {Object.entries(allPermissions).map(([action, { description, permission }]) => (
                      <PermissionToggle
                        key={action}
                        permissionName={permission}
                        label={action}
                        description={description}
                        checked={selectedPermissions.includes(permission)}
                        setChecked={(isChecked) => handleSetChecked(permission, isChecked)}
                      />
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>
      {apis.map((api) => (
        <Collapsible
          key={api.id}
          open={cardStatesMap[api.id]}
          onOpenChange={() => {
            toggleCard(api.id);
          }}
        >
          <Card>
            <CardHeader>
              <CollapsibleTrigger
                className="flex items-center justify-between transition-all pb-6 [&[data-state=open]>svg]:rotate-90"
                aria-controls={api.id}
                aria-expanded={cardStatesMap[api.id]}
              >
                <CardTitle className="break-all">{api.name}</CardTitle>
                <ChevronRight
                  className="w-4 h-4 transition-transform duration-200"
                  aria-hidden="true"
                />
              </CollapsibleTrigger>
              <CollapsibleContent id={api.id}>
                <CardDescription>
                  Permissions scoped to this API. Enabling these roles only grants access to this
                  specific API.
                </CardDescription>
              </CollapsibleContent>
            </CardHeader>
            <CollapsibleContent>
              <CardContent>
                <div className="flex flex-col gap-4">
                  {Object.entries(apiPermissions(api.id)).map(([category, roles]) => {
                    const allPermissionNames = Object.values(roles).map(
                      ({ permission }) => permission,
                    );
                    const isAllSelected = allPermissionNames.every((permission) =>
                      selectedPermissions.includes(permission),
                    );

                    return (
                      <div key={`api-${category}`} className="flex flex-col gap-2">
                        <div className="flex flex-col">
                          <PermissionToggle
                            permissionName={`selectAll-${category}`}
                            label={<span className="text-base font-bold">{category}</span>}
                            description={`Select all ${category} permissions for this API`}
                            checked={isAllSelected}
                            setChecked={(isChecked) => {
                              allPermissionNames.forEach((permission) => {
                                handleSetChecked(permission, isChecked);
                              });
                            }}
                          />
                        </div>

                        <div className="flex flex-col gap-1">
                          {Object.entries(roles).map(([action, { description, permission }]) => (
                            <PermissionToggle
                              key={action}
                              permissionName={permission}
                              label={action}
                              description={description}
                              checked={selectedPermissions.includes(permission)}
                              setChecked={(isChecked) => handleSetChecked(permission, isChecked)}
                            />
                          ))}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </CollapsibleContent>
          </Card>
        </Collapsible>
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
            setName("");
            router.push("/settings/root-keys");
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
              <div className="flex items-center justify-between gap-2">
                <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
                <CopyButton value={key.data?.key ?? ""} />
              </div>
            </Code>
          </DialogHeader>

          <p className="mt-2 text-sm font-medium text-center text-gray-700 ">
            Try creating a new api key for your users:
          </p>
          <Code className="flex flex-col items-start gap-2 w-full text-xs">
            <div className="w-full shrink-0 flex items-center justify-end gap-2">
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
              <CopyButton value={snippet} />
            </div>
            <div className="text-wrap">
              {showKeyInSnippet ? snippet : snippet.replace(key.data?.key ?? "", maskedKey)}
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
  label: string | React.ReactNode;
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
    <div className="flex items-center gap-0">
      <div className="w-4/6 mr-2 md:w-1/3">
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
