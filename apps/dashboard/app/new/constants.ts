export const KEY_PARAM = "key";
export const API_ID_PARAM = "apiId";

export type StepInfo = {
  title: string;
  description: string;
};

export const stepInfos: StepInfo[] = [
  {
    title: "Create Company Workspace",
    description: "Customize your workspace name. This is how it’ll appear in your dashboard.",
  },
  {
    title: "Create Your First API Key",
    description:
      "Generate a key for your public API. You’ll be able to verify, revoke, and track usage — all globally distributed with built-in analytics.",
  },
  {
    title: "Your API Key is Ready",
    description:
      "Use the code snippet below to start authenticating requests. You can always manage or rotate your keys from the dashboard.",
  },
];
