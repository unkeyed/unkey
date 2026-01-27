import type { DeploymentNode } from "../types";

export const SKELETON_TREE: DeploymentNode = {
  id: "internet",
  label: "INTERNET",
  direction: "horizontal",
  metadata: { type: "origin" },
  children: [
    {
      id: "us-east-1-skeleton",
      label: "us-east-1",
      direction: "vertical",
      metadata: { type: "skeleton" },
      children: [
        {
          id: "us-east-1-s-1-skeleton",
          label: "s-skeleton-1",
          metadata: { type: "skeleton" },
        },
        {
          id: "us-east-1-s-2-skeleton",
          label: "s-skeleton-2",
          metadata: { type: "skeleton" },
        },
      ],
    },
    {
      id: "eu-central-1-skeleton",
      label: "eu-central-1",
      direction: "vertical",
      metadata: { type: "skeleton" },
      children: [
        {
          id: "eu-central-1-s-1-skeleton",
          label: "s-skeleton-1",
          metadata: { type: "skeleton" },
        },
        {
          id: "eu-central-1-s-2-skeleton",
          label: "s-skeleton-2",
          metadata: { type: "skeleton" },
        },
        {
          id: "eu-central-1-s-3-skeleton",
          label: "s-skeleton-3",
          metadata: { type: "skeleton" },
        },
      ],
    },
    {
      id: "ap-southeast-2-skeleton",
      label: "ap-southeast-2",
      direction: "vertical",
      metadata: { type: "skeleton" },
      children: [
        {
          id: "ap-southeast-2-s-1-skeleton",
          label: "s-skeleton-1",
          metadata: { type: "skeleton" },
        },
        {
          id: "ap-southeast-2-s-2-skeleton",
          label: "s-skeleton-2",
          metadata: { type: "skeleton" },
        },
      ],
    },
  ],
};
