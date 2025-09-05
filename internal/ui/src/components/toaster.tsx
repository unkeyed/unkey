"use client";
import { useTheme } from "next-themes";
import type React from "react";
import { Toaster as Sonner, toast } from "sonner";

type ToasterProps = React.ComponentProps<typeof Sonner>;

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme } = useTheme();

  return (
    <Sonner
      theme={theme === "light" || theme === "dark" || theme === "system" ? theme : "system"}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-background group-[.toaster]:text-content group-[.toaster]:border-border group-[.toaster]:shadow-lg",
          description: "group-[.toast]:!text-gray-11",
          actionButton: "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton: "group-[.toast]:bg-secondary group-[.toast]:text-content-subtle",
        },
      }}
      {...props}
    />
  );
};

export { Toaster, toast };
