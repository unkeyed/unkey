import { CallbackShell } from "./callback-shell";

export const dynamic = "force-dynamic";
export const revalidate = 0;

export default function Layout({ children }: { children: React.ReactNode }) {
  return <CallbackShell>{children}</CallbackShell>;
}
