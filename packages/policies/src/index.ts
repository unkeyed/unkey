import { Policy as GenericPolicy } from "@chronark/access-policies";

export type Resources = {
  api: [
    /**
     * Create a new api
     */
    "create",
    /**
     * Read an api's configuration
     */
    "read",
    /**
     * Change an existing api's configuration
     */
    "update",
    /**
     * Can delete an api and all its keys
     */
    "delete",
    "read:keys",
    "create:keys",
  ];
};

type TenantId = string;
type ResourceId = string;
/**
 * Global Resource ID
 */
export type GRID = `${TenantId}::${keyof Resources | "*"}::${ResourceId}`;

export class Policy extends GenericPolicy<Resources, GRID> {}
