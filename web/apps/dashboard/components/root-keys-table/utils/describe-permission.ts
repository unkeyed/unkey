const ACRONYMS: Record<string, string> = {
  api: "API",
  apis: "APIs",
  github: "GitHub",
  id: "ID",
  ids: "IDs",
  rbac: "RBAC",
  url: "URL",
};

export function humaniseAction(action: string): string {
  return action
    .split("_")
    .map((word, index) => {
      const acronym = ACRONYMS[word];
      if (acronym) {
        return acronym;
      }
      return index === 0 ? word.charAt(0).toUpperCase() + word.slice(1) : word;
    })
    .join(" ");
}

// Legacy root key permissions are stored as `<resource>.<instance>.<action>`
// (see `unkeyPermissionValidation` in @unkey/rbac). Only the action carries
// meaning for a reader; the instance is always `*` or an id they cannot read.
export function describePermission(name: string): string {
  if (name === "*") {
    return "All permissions";
  }

  const action = name.split(".").at(-1);
  if (!action) {
    return name;
  }

  return humaniseAction(action);
}

export function describePermissions(names: readonly string[]): string[] {
  return [...new Set(names.map(describePermission))];
}
