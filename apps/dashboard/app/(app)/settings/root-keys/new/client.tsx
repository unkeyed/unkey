"use client";

import { ConfirmPopover } from "@/components/confirmation-popover";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { trpc } from "@/lib/trpc/client";
import { Check, CircleInfo, Key2 } from "@unkey/icons";
import { type UnkeyPermission, unkeyPermissionValidation } from "@unkey/rbac";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Checkbox,
  Input,
  toast,
} from "@unkey/ui";
import { ChevronRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { createParser, parseAsArrayOf, useQueryState } from "nuqs";
import { useEffect, useRef, useState } from "react";
import { SecretKey } from "../../../apis/[apiId]/_components/create-key/components/secret-key";
import { apiPermissions, workspacePermissions } from "../[keyId]/permissions/permissions";

type Props = {
  apis: {
    id: string;
    name: string;
  }[];
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

const parseAsUnkeyPermission = createParser({
  parse(queryValue) {
    const { success, data } = unkeyPermissionValidation.safeParse(queryValue);
    return success ? data : null;
  },
  serialize: String,
});

export const Client: React.FC<Props> = ({ apis }) => {
  const trpcUtils = trpc.useUtils();
  const router = useRouter();
  const [name, setName] = useState<string | undefined>(undefined);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<
    "close" | "create-another" | "go-to-details" | null
  >(null);
  const dividerRef = useRef<HTMLDivElement>(null);

  const [selectedPermissions, setSelectedPermissions] = useQueryState(
    "permissions",
    parseAsArrayOf(parseAsUnkeyPermission).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );

  const key = trpc.rootKey.create.useMutation({
    onSuccess() {
      trpcUtils.settings.rootKeys.query.invalidate();
    },
    onError(err: { message: string }) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const handleSetChecked = (permission: UnkeyPermission, checked: boolean) => {
    setSelectedPermissions((prevPermissions) => {
      const permissionSet = new Set(prevPermissions);

      if (checked) {
        permissionSet.add(permission);
      } else {
        permissionSet.delete(permission);
      }

      return Array.from(permissionSet);
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

  const handleCloseAttempt = (action: "close" | "create-another" | "go-to-details" = "close") => {
    setPendingAction(action);
    setIsConfirmOpen(true);
  };

  const handleConfirmClose = () => {
    if (!pendingAction) {
      console.error("No pending action when confirming close");
      return;
    }

    setIsConfirmOpen(false);

    try {
      // Always close the dialog first
      key.reset();
      setSelectedPermissions([]);
      setName("");

      // Then execute the specific action
      switch (pendingAction) {
        case "create-another":
          // Reset form for creating another key
          break;

        case "go-to-details":
          router.push("/settings/root-keys");
          break;

        default:
          // Dialog already closed, nothing more to do
          router.push("/settings/root-keys");
          break;
      }
    } catch (error) {
      console.error("Error executing pending action:", error);
      toast.error("Action Failed", {
        description: "An unexpected error occurred. Please try again.",
      });
    } finally {
      setPendingAction(null);
    }
  };

  const handleDialogOpenChange = (open: boolean) => {
    if (!open) {
      handleCloseAttempt("close");
    }
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
                      label={<span className="text-base font-bold">{category}</span>}
                      description={`Select all permissions for ${category} in this workspace`}
                      checked={isAllSelected}
                      setChecked={(isChecked: boolean) => {
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
                        setChecked={(isChecked: boolean) => handleSetChecked(permission, isChecked)}
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
                            setChecked={(isChecked: boolean) => {
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
                              setChecked={(isChecked: boolean) =>
                                handleSetChecked(permission, isChecked)
                              }
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
        Create New Key
      </Button>

      <Dialog open={!!key.data?.key} onOpenChange={handleDialogOpenChange}>
        <DialogContent
          className="drop-shadow-2xl border-gray-4 overflow-hidden !rounded-2xl p-0 gap-0 min-w-[760px] max-h-[90vh] overflow-y-auto"
          showCloseWarning
          onAttemptClose={() => handleCloseAttempt("close")}
        >
          <>
            <div className="bg-grayA-2 py-10 flex flex-col items-center justify-center w-full px-[120px]">
              <div className="py-4 mt-[30px]">
                <div className="flex gap-4">
                  <div className="border border-grayA-4 rounded-[10px] size-14 opacity-35" />
                  <div className="border border-grayA-4 rounded-[10px] size-14" />
                  <div className="border border-grayA-4 rounded-[10px] size-14 flex items-center justify-center relative">
                    <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute left-0 top-0" />
                    <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute right-0 top-0" />
                    <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute right-0 bottom-0" />
                    <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute left-0 bottom-0" />
                    <Key2 size="2xl-thin" />
                    <div className="flex items-center justify-center border border-grayA-3 rounded-full bg-success-9 text-white size-[22px] absolute right-[-10px] top-[-10px]">
                      <Check size="sm-bold" />
                    </div>
                  </div>
                  <div className="border border-grayA-4 rounded-[10px] size-14" />
                  <div className="border border-grayA-4 rounded-[10px] size-14 opacity-35" />
                </div>
              </div>
              <div className="mt-5 flex flex-col gap-2 items-center">
                <div className="font-semibold text-gray-12 text-[16px] leading-[24px]">
                  Root Key Created
                </div>
                <div
                  className="text-gray-10 text-[13px] leading-[24px] text-center"
                  ref={dividerRef}
                >
                  You've successfully generated a new Root key.
                </div>
              </div>
              <div className="p-1 w-full my-8">
                <div className="h-[1px] bg-grayA-3 w-full" />
              </div>

              <div className="flex flex-col gap-2 items-start w-full">
                <div className="text-gray-12 text-sm font-semibold">Root Key</div>
                <SecretKey
                  value={key.data?.key ?? ""}
                  title="API Key"
                  className="bg-white dark:bg-black "
                />
                <div className="text-gray-9 text-[13px] flex items-center gap-1.5">
                  <CircleInfo className="text-accent-9" size="sm-regular" />
                  <span>
                    Copy and save this secret as it won't be shown again.{" "}
                    <a
                      href="https://www.unkey.com/docs/security/recovering-keys"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-info-11 hover:underline"
                    >
                      Learn more
                    </a>
                  </span>
                </div>
              </div>
              <div className="mt-6">
                <div className="mt-4 text-center text-gray-10 text-xs leading-6">
                  All set! You can now create another root key or explore the docs to learn more
                </div>
              </div>
            </div>
            <ConfirmPopover
              isOpen={isConfirmOpen}
              onOpenChange={setIsConfirmOpen}
              onConfirm={handleConfirmClose}
              triggerRef={dividerRef}
              title="You won't see this secret key again!"
              description="Make sure to copy your secret key before closing. It cannot be retrieved later."
              confirmButtonText="Close anyway"
              cancelButtonText="Dismiss"
              variant="warning"
              popoverProps={{
                side: "right",
                align: "end",
                sideOffset: 5,
                alignOffset: 30,
                onOpenAutoFocus: (e) => e.preventDefault(),
              }}
            />
          </>
        </DialogContent>
      </Dialog>
    </div>
  );
};
