import { ReactNode } from "react";

interface TeamLayoutProps {
  children: ReactNode;
  modal: ReactNode;
}

export default function TeamLayout({ children, modal }: TeamLayoutProps) {
  return (
    <>
      {children}
      {modal}
    </>
  );
}
