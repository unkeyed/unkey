import { apiPermissions, workspacePermissions } from "../../../[keyId]/permissions/permissions";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Badge, Button } from "@unkey/ui";
import { ChevronDown, XMark } from "@unkey/icons"
import type { UnkeyPermission } from "@unkey/rbac";
import { cn } from "@/lib/utils";

type Props = {
    apiId: string;
    name: string;
    selectedPermissions: UnkeyPermission[];
    expandCount: number;
    title: string;
    removePermission: (permission: string) => void;
};

type InfoType = { permission: UnkeyPermission, category: string, action: string }[]
const PermissionBadgeList = ({ apiId, name, selectedPermissions, title, expandCount, removePermission }: Props) => {
   
    const workspace = workspacePermissions;
    const allPermissions = apiId === "workspace" ? workspace : apiPermissions(apiId);

    // Flatten allPermissions into an array of {permission, action} objects
    const allPermissionsArray = Object.entries(allPermissions).flatMap(([category, permissions]) =>
        Object.entries(permissions).map(([action, permissionData]) => ({
            permission: permissionData.permission,
            category,
            action,
        }))
    );
    const info = findPermission(allPermissionsArray, selectedPermissions);
    if (info.length === 0) return null;
    return info.length > expandCount ? (
        <div className="flex flex-col gap-2">
            <ListTitle title={title} count={info.length} category={name} />
            <CollapsibleList info={info as InfoType} title={title} expandCount={expandCount} removePermission={removePermission} />
        </div>
    ) : (
        <div className="flex flex-col gap-2">
            <ListTitle title={title} count={info.length} category={name} />
            <ListBadges info={info as InfoType} removePermission={removePermission} />
        </div>
    )

};

const findPermission = (allPermissions: InfoType, selectedPermissions: string[]) => {
    return selectedPermissions.map((permission) => {
        return allPermissions.find((p) => p.permission === permission);
    }).filter(Boolean);
}
const ListBadges = ({ info, removePermission }: { info: InfoType, removePermission: (permission: string) => void }) => {
    const handleRemovePermission = (e: React.MouseEvent<HTMLButtonElement>, permission: string) => {
        e.stopPropagation();
        removePermission(permission);
    }
    return (
        <div className="flex flex-wrap gap-2">
            {info?.map((permission) => {
                if (!permission) return null;
                return (
                    <Badge key={permission.permission} variant="primary" className="flex items-center h-[22px] p-[6px] px-2 text-xs font-normal text-grayA-11 hover:bg-grayA-2 hover:text-grayA-12 gap-2">
                        <span>{permission.action}</span>
                        <Button variant="ghost" size="icon" className="w-4 h-4" onClick={(e) => handleRemovePermission(e, permission.permission)}>
                            <XMark className="w-4 h-4" />
                        </Button>
                    </Badge>
                )
            })}
        </div>
    )
}

const CollapsibleList = ({ info, title, expandCount, removePermission }: { info: InfoType, title: string, expandCount: number, removePermission: (permission: string) => void }) => {
    const infoFirst = info.slice(0, expandCount) as InfoType;
    const infoRest = info.slice(expandCount) as InfoType;
    return (
        <Collapsible>
            <CollapsibleTrigger
                className={cn(
                    "flex flex-col gap-3 transition-all [&[data-state=open]>svg]:rotate-180 w-full",
                )}
            >
                <div className="flex flex-row items-center w-full">
                    <ListBadges info={infoFirst} removePermission={removePermission} />
                    <ChevronDown className="w-4 h-4 transition-transform duration-200 ml-auto text-grayA-8" />
                </div>
            </CollapsibleTrigger>
            <CollapsibleContent>
                <div className="flex flex-wrap gap-2 pt-2">
                    <ListBadges info={infoRest} removePermission={removePermission} />
                </div>
            </CollapsibleContent>
        </Collapsible>
    )
}

// const ListTitleBadges = ({ info }: { info: InfoType }) => {
//     return (
//         <div className="flex flex-wrap gap-2">
//             {info?.map((permission) => {
//                 if (!permission) return null;
//                 return (
//                     <Badge key={permission.permission}
//                         variant="primary"
//                         className="flex items-center h-[22px] p-[6px] px-2 text-xs font-normal text-grayA-11 hover:bg-grayA-2 hover:text-grayA-12 gap-2">
//                         <span>{permission.action}</span>
//                         <XMark className="w-4 h-4" />
//                     </Badge>
//                 )
//             })}
//         </div>
//     )
// }
const ListTitle = ({ title, count, category }: { title: string, count: number, category: string }) => {
    return (
        <p className="text-sm w-full text-grayA-10 justify-start text-left">
            {title}<span className="font-bold text-gray-11 ml-2">{category}</span>
            <Badge variant="primary"
                size="sm"
                className="text-[11px] font-normal text-grayA-11 rounded-full px-2 ml-4 h-[18px] min-w-[22px] border-[1px] border-grayA-3 ">
                {count}
            </Badge>
        </p>
    )
}
export { PermissionBadgeList };