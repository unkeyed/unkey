import { Policy as GenericPolicy } from "@chronark/access-policies";

export type Resources = {
  api: [
    /**
     * Create a new api
     */
    "create",
    /**
     * Read api config
     */
    "read",
    /**
     * Change an existing api's configuration
     */
    "update",
    /**
     * Can delete an api
     */
    "delete",
    "create:key",
  ];
  policy: ["create", "read", "update", "delete"];
  key: ["create", "read", "update", "delete", "attach:policy", "detach:policy"];
};

type TenantId = string;
type ResourceId = string; // a specific apiId or keyId for example
/**
 * Global Resource ID
 */
export type GRID =
  | `${TenantId}::${keyof Resources}::${ResourceId | "*"}` // standard
  | `${TenantId}::api::${ResourceId}::${Exclude<keyof Resources, "api"> | "*"}::${
      | ResourceId
      | "*"}`; // api specific nesting

export class Policy extends GenericPolicy<Resources, GRID> {
  toJSON() {
    return JSON.parse(this.toString());
  }

  static fromJSON(json: unknown) {
    return Policy.parse(JSON.stringify(json));
  }
}
