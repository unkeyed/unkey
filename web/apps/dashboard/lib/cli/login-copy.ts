import type { DeviceLogin } from "./device-api";

export type LoginCopy = {
  title: string;
  description: string;
  confirmCode: boolean;
  accessDescription: string;
  noPermissions: string;
  warning: string;
  authorizeLabel: string;
  approvedTitle: string;
  approvedDescription: string;
  deniedDescription: string;
  expiredDescription: string;
  docs: { label: string; href: string };
};

const cliCopy: LoginCopy = {
  title: "Sign in to the Unkey CLI",
  description: "Approve this request so the CLI can work in your workspace.",
  confirmCode: true,
  accessDescription: "Pick what the CLI can do in this workspace.",
  noPermissions: "Your role has no permissions it can give to the CLI.",
  warning:
    "Only authorize if you started this login. Whoever started it gets access to this workspace.",
  authorizeLabel: "Authorize CLI",
  approvedTitle: "You're signed in to the Unkey CLI",
  approvedDescription: "Head back to your terminal to keep going. You can close this tab.",
  deniedDescription: "Run unkey login again if you want to start a new one.",
  expiredDescription: "Run unkey login again to get a new code.",
  docs: { label: "CLI docs", href: "https://www.unkey.com/docs/platform/cli/overview" },
};

export function loginCopy(login: Pick<DeviceLogin, "oauth">): LoginCopy {
  if (!login.oauth) {
    return cliCopy;
  }
  const { clientName, clientHost, resource } = login.oauth;
  const server = serverLabel(resource);
  return {
    title: `Authorize ${clientName}`,
    description: `${clientName} (${clientHost}) wants to use Unkey through ${server}.`,
    confirmCode: false,
    accessDescription: `Pick what ${clientName} can do in this workspace.`,
    noPermissions: `Your role has no permissions it can give to ${clientName}.`,
    warning: `Only authorize if you just connected ${clientName} to Unkey. It gets access to this workspace.`,
    authorizeLabel: "Authorize",
    approvedTitle: `Authorized ${clientName}`,
    approvedDescription: `Returning you to ${clientName}. You can close this tab.`,
    deniedDescription: `${clientName} did not get access to this workspace.`,
    expiredDescription: `Connect again from ${clientName}.`,
    docs: { label: "MCP docs", href: "https://www.unkey.com/docs/platform/ai-coding/mcp-servers" },
  };
}

export function serverLabel(resource: string): string {
  return resource.replace(/^https?:\/\//, "");
}
