"use client";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Api, Permission } from "@unkey/db";
import { type PropsWithChildren, useMemo, useState } from "react";
import { PermissionToggle } from "./permission_toggle";
import { apiPermissions } from "./permissions";

export function DialogAddPermissionsForAPI(
  props: PropsWithChildren<{
    keyId: string;
    apis: { id: string; name: string }[];
    permissions: Permission[];
  }>,
) {
  const apisWithoutPermission = props.apis.filter((api) => {
    const apiPermissionsStructure = apiPermissions(api.id);
    const hasActivePermissions = Object.entries(apiPermissionsStructure).some(
      ([_category, allPermissions]) => {
        const amountActiveRules = Object.entries(allPermissions).filter(
          ([_action, { description: _description, permission }]) => {
            return props.permissions.some((p) => p.name === permission);
          },
        );

        return amountActiveRules.length > 0;
      },
    );

    return !hasActivePermissions;
  });

  const [selectedApiId, setSelectedApiId] = useState<string>("");
  const selectedApi = useMemo(
    () => props.apis.find((api) => api.id === selectedApiId),
    [selectedApiId],
  );

  const isSelectionDisabled =
    selectedApi && !apisWithoutPermission.some((api) => api.id === selectedApi.id);

  const options = apisWithoutPermission.reduce(
    (map, api) => {
      map[api.id] = api.name;
      return map;
    },
    {} as Record<string, string>,
  );

  function onOpenChange() {
    setSelectedApiId("");
  }

  return (
    <Dialog onOpenChange={onOpenChange}>
      {/* Trigger should be in here */}
      {props.children}

      <DialogContent className="sm:max-w-[640px] max-h-[70vh] overflow-y-scroll">
        <DialogHeader>
          <DialogTitle>Setup permissions for an API</DialogTitle>
          <Select
            value={selectedApiId}
            onValueChange={setSelectedApiId}
            disabled={isSelectionDisabled}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select an API" />
            </SelectTrigger>
            <SelectContent>
              {Object.entries(options).map(([id, label]) => (
                <SelectItem key={id} value={id}>
                  {label}
                </SelectItem>
              ))}
              {selectedApi && !Object.entries(options).some(([id]) => id === selectedApiId) && (
                <SelectItem value={selectedApiId}>{selectedApi.name}</SelectItem>
              )}
            </SelectContent>
          </Select>
        </DialogHeader>

        {selectedApiId !== "" && (
          <div className="flex flex-col w-full gap-4">
            {Object.entries(apiPermissions(selectedApiId)).map(([category, allPermissions]) => (
              <div className="flex flex-col gap-2">
                <span className="font-medium">{category}</span>{" "}
                <div className="flex flex-col gap-1">
                  {Object.entries(allPermissions).map(([action, { description, permission }]) => {
                    return (
                      <PermissionToggle
                        key={action}
                        rootKeyId={props.keyId}
                        permissionName={permission}
                        label={action}
                        description={description}
                        checked={props.permissions.some((p) => p.name === permission)}
                        preventDisabling={!selectedApi}
                        preventEnabling={!selectedApi}
                      />
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        )}

        {/* <DialogFooter>
        <Button type="submit">
          {createRole.isLoading ? <Loading className="w-4 h-4" /> : "Create"}
        </Button>
      </DialogFooter> */}
      </DialogContent>
    </Dialog>
  );
}

// {
//   apisWithoutActivePermissions.length > 0 && (
//     <Card className="flex w-full items-center justify-center h-36 border-dashed">
//       <DialogTrigger asChild>
//         <Button variant="outline">
//           Add permissions for {apisWithActivePermissions.length > 0 ? "another" : "an"} API
//         </Button>
//       </DialogTrigger>
//     </Card>
//   );
// }

// type PermissionToggleProps = {
//   rootKeyId: string;
//   permissionName: string;
//   label: string;
//   description: string;
//   checked: boolean;
//   preventEnabling?: boolean;
//   preventDisabling?: boolean;
// };

// export const PermissionToggle: React.FC<PermissionToggleProps> = ({
//   rootKeyId,
//   permissionName,
//   label,
//   checked,
//   description,
//   preventEnabling,
//   preventDisabling,
// }) => {
//   const router = useRouter();

//   const [optimisticChecked, setOptimisticChecked] = useState(checked);
//   const addPermission = trpc.rbac.addPermissionToRootKey.useMutation({
//     onMutate: () => {
//       setOptimisticChecked(true);
//     },
//     onSuccess: () => {
//       toast.success("Permission added", {
//         description: "Changes may take up to 60 seconds to take effect.",
//       });
//     },
//     onError: (error) => {
//       toast.error(error.message);
//     },
//     onSettled: () => {
//       router.refresh();
//     },
//   });
//   const removeRole = trpc.rbac.removePermissionFromRootKey.useMutation({
//     onMutate: () => {
//       setOptimisticChecked(false);
//     },
//     onSuccess: () => {
//       toast.success("Permission removed", {
//         description: "Changes may take up to 60 seconds to take effect.",
//         cancel: {
//           label: "Undo",
//           onClick: () => {
//             addPermission.mutate({ rootKeyId, permission: permissionName });
//           },
//         },
//       });
//     },
//     onError: (error) => {
//       toast.error(error.message);
//     },
//     onSettled: () => {
//       router.refresh();
//     },
//   });

//   return (
//     <div className="@container flex items-center text-start">
//       <div className="flex flex-col items-center @xl:flex-row w-full">
//         <div className="w-full @xl:w-1/3">
//           <Tooltip>
//             <TooltipTrigger className="flex items-center gap-2">
//               {addPermission.isLoading || removeRole.isLoading ? (
//                 <Loader2 className="w-4 h-4 animate-spin" />
//               ) : (
//                 <Checkbox
//                   disabled={
//                     addPermission.isLoading ||
//                     removeRole.isLoading ||
//                     (preventEnabling && !checked) ||
//                     (preventDisabling && checked)
//                   }
//                   checked={optimisticChecked}
//                   onClick={() => {
//                     if (checked) {
//                       if (!preventDisabling) {
//                         removeRole.mutate({ rootKeyId, permissionName });
//                       }
//                     } else {
//                       if (!preventEnabling) {
//                         addPermission.mutate({ rootKeyId, permission: permissionName });
//                       }
//                     }
//                   }}
//                 />
//               )}
//               <Label className="text-xs text-content">{label}</Label>
//             </TooltipTrigger>
//             <TooltipContent className="flex items-center gap-2">
//               <span className="font-mono text-sm font-medium">{permissionName}</span>
//               <CopyButton value={permissionName} />
//             </TooltipContent>
//           </Tooltip>
//         </div>

//         <p className="w-full text-xs text-content-subtle @xl:w-2/3 pl-8 pb-2 @xl:pb-0 @xl:mt-0">
//           {description}
//         </p>
//       </div>
//     </div>
//   );
// };
