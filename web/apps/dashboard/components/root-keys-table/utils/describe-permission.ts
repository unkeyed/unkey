// Root key permissions use either the legacy `<resource>.<instance>.<action>` format or v2 URNs.
// Only the action segment carries meaning for this label.
const ACRONYMS: Record<string, string> = {
  api: "API",
  apis: "APIs",
  github: "GitHub",
  id: "ID",
  ids: "IDs",
  rbac: "RBAC",
  url: "URL",
};

export function describePermission(name: string): string {
  if (name === "*") {
    return "All permissions";
  }

  const action = name.includes("#") ? name.split("#").at(-1) : name.split(".").at(-1);
  if (!action) {
    return name;
  }

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

export function describePermissions(names: readonly string[]): string[] {
  return [...new Set(names.map(describePermission))];
}
