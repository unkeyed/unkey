export type StepInfo = {
  title: string;
  description: string;
};

export const stepInfos: StepInfo[] = [
  {
    title: "Create company workspace",
    description:
      "Customize your workspace name, logo, and handle. This is how it’ll appear in your dashboard and URLs.",
  },
  {
    title: "Create your first API key",
    description:
      "Generate a key for your public API. You’ll be able to verify, revoke, and track usage — all globally distributed with built-in analytics.",
  },
  {
    title: "Configure your dashboard",
    description: "Customize your dashboard settings and invite team members to collaborate.",
  },
];
