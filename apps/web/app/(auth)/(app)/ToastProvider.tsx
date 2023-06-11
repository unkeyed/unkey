"use client";

import React, { PropsWithChildren } from "react";

import {
  ToastProvider as Provider,
  Toast,
  ToastTitle,
  ToastDescription,
  ToastClose,
  ToastViewport,
} from "@/components/ui/toast";
import { useToast } from "@/components/ui/use-toast";

export const ToastProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const { toasts } = useToast();
  return (
    <Provider>
      {toasts.map(function ({ id, title, description, action, ...props }) {
        return (
          <Toast key={id} {...props}>
            <div className="grid gap-1">
              {title && <ToastTitle>{title}</ToastTitle>}
              {description && <ToastDescription>{description}</ToastDescription>}
            </div>
            {action}
            <ToastClose />
          </Toast>
        );
      })}
      <ToastViewport />
      {children}
    </Provider>
  );
};
