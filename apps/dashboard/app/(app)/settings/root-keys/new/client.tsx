"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Loading } from "@/components/dashboard/loading";
import { VisibleButton } from "@/components/dashboard/visible-button";
import { Code } from "@/components/ui/code";
import { Button } from "@unkey/ui";

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
import { trpc } from "@/lib/trpc/client";
import { type UnkeyPermission, unkeyPermissionValidation } from "@unkey/rbac";
import { ChevronRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { createParser, parseAsArrayOf, useQueryState } from "nuqs";
import { useEffect, useState } from "react";
import { apiPermissions, workspacePermissions } from "../[keyId]/permissions/permissions";

type Props = {
  apis: {
    id: string;
    name: string;
  }[];
};

const parseAsUnkeyPermission = createParser({
  parse(queryValue) {
    const { success, data } = unkeyPermissionValidation.safeParse(queryValue);
    return success ? data : null;
  },
  serialize: String,
});

export const Client: React.FC<Props> = ({ apis }) => {
  const router = useRouter();
  const [name, setName] = useState<string | undefined>(undefined);

  const [selectedPermissions, setSelectedPermissions] = useQueryState(
    "permissions",
    parseAsArrayOf(parseAsUnkeyPermission).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );

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

  const [cardStatesMap, setCardStatesMap] = useState<Record<string, boolean>>({});

  const toggleCard = (apiId: string) => {
    setCardStatesMap((prevStates) => ({
      ...prevStates,
      [apiId]: !prevStates[apiId],
    }));
  };

  // biome-ignore lint/correctness/useExhaustiveDependencies: effect must be called once to set initial cards state
  useEffect(() => {
    const initialSelectedApiSet = new Set<string>();
    selectedPermissions.forEach((permission) => {
      const apiId = permission.split(".")[1] ?? ""; // Extract API ID
      if (apiId.length) {
        initialSelectedApiSet.add(apiId);
      }
    });

    const initialCardStates: Record<string, boolean> = {};
    apis.forEach((api) => {
      initialCardStates[api.id] = initialSelectedApiSet.has(api.id); // O(1) check
    });

    // We use a Set to gather unique API IDs, enabling O(1) membership checks.
    // This avoids the O(m * n) complexity of repeatedly iterating over selectedPermissions
    // for each API, reducing the overall complexity to O(n + m) and improving performance
    // for large data sets.

    setCardStatesMap(initialCardStates);
  }, []); // Execute ones on the first load

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
                            label={
                              <span className="text-base font-bold mt-3.5 sm:mt-0">{category}</span>
                            }
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
        <DialogContent className="flex flex-col max-sm:w-full bg-grayA-1 border-gray-4">
          <DialogHeader>
            <DialogTitle>Your API Key</DialogTitle>
            <DialogDescription className="w-fit text-accent-10">
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

          <p className="mt-2 text-sm font-medium text-center text-accent-10">
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
  label: string | React.ReactNode;
  description: string;
};

const PermissionToggle: React.FC<PermissionToggleProps> = ({
  checked,
  setChecked,
  label,
  description,
}) => {
  return (
    <div className="flex flex-col sm:items-center gap-1 mb-2 sm:flex-row sm:gap-0 sm:mb-0">
      <div className="w-1/3 flex items-center gap-2">
        <Checkbox
          checked={checked}
          onCheckedChange={() => {
            setChecked(!checked);
          }}
        />
        <Label className="text-xs text-content">{label}</Label>
      </div>

      <p className="w-full md:w-2/3 text-xs text-content-subtle ml-6 md:ml-0">{description}</p>
    </div>
  );
};
