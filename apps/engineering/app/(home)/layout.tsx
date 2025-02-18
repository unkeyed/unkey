import { HomeLayout } from "fumadocs-ui/layouts/home";
import type { ReactNode } from "react";
import { baseOptions } from "../layout.config";

export default function Layout({
  children,
}: {
  children: ReactNode;
}): React.ReactElement {
  return <HomeLayout {...baseOptions}>{children}</HomeLayout>;
}
