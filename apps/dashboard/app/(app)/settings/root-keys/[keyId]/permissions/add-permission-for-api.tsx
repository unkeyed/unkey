"use client";

import { DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Api, Permission } from "@unkey/db";
import { useState } from "react";
import { PermissionToggle } from "./permission_toggle";
import { apiPermissions } from "./permissions";
import { Button } from "@/components/ui/button";
import { Loading } from "@/components/dashboard/loading";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { trpc } from "@/lib/trpc/client";
import { Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";

export function DialogContentAddPermissionsForAPI(props: {
  keyId: string;
  apisWithoutActivePermissions: Api[];
  permissions: Permission[];
}) {
  const [apiId, setApiId] = useState<string>("");

  const options = props.apisWithoutActivePermissions.reduce((map, api) => {
    map[api.id] = api.name;
    return map;
  }, {});

  return (
    <DialogContent className="sm:max-w-[640px] max-h-[70vh] overflow-y-scroll">
      <DialogHeader>
        <DialogTitle>Setup permissions for an API</DialogTitle>
        <Select value={apiId} onValueChange={setApiId}>
          <SelectTrigger>
            <SelectValue defaultValue={apiId} />
          </SelectTrigger>
          <SelectContent>
            {Object.entries(options).map(([id, label]) => (
              <SelectItem key={id} value={id}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </DialogHeader>

      {apiId !== null && (
        <div className="flex flex-col w-full gap-4">
          {Object.entries(apiPermissions(apiId)).map(([category, allPermissions]) => (
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
  );
}

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
