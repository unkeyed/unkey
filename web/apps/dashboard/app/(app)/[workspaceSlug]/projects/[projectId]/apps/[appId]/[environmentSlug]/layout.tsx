import type { PropsWithChildren } from "react";
import { AppEnvironmentProvider } from "./environment-context";

export default function EnvironmentLayout({ children }: PropsWithChildren) {
  return <AppEnvironmentProvider>{children}</AppEnvironmentProvider>;
}
