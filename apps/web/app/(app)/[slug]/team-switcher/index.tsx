import { auth, getUserId } from "@/lib/auth";
import { db } from "@/lib/db";
import { headers } from "next/headers";
import { Menu } from "./menu";

type Props = {
  slug: string;
};

export const WorkspaceSwitcher: React.FC<Props> = async ({ slug }) => {
  const userId = await getUserId();
  const workspaces = await auth.listWorkspaces(userId);

  console.log({ workspaces });

  return <Menu current={workspaces.find((ws) => ws.slug === slug)!} workspaces={workspaces} />;
};
