import { ReactNode } from "react";

interface TeamLayoutProps {
  children: ReactNode;
}

export default function TeamLayout({ children }: TeamLayoutProps) {
  return <>{children}</>;
}
