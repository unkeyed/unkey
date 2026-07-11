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
            "group toast group-[.toaster]:bg-white dark:group-[.toaster]:bg-black group-[.toaster]:text-gray-12 group-[.toaster]:border-border group-[.toaster]:shadow-lg",
          description: "group-[.toast]:text-gray-11!",
          actionButton: "group-[.toast]:bg-gray-12 group-[.toast]:text-gray-1",
          cancelButton: "group-[.toast]:bg-gray-3 group-[.toast]:text-gray-11",
        },
      }}
      {...props}
    />
  );
};

export { Toaster, toast };
