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
