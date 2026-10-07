"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRef } from "react";
import { useForm } from "react-hook-form";
import { type RootKeyFormValues, rootKeySchema } from "../schema";

export function useRootKeyPolicyForm(
  defaultValues: RootKeyFormValues,
  onValid: (values: RootKeyFormValues) => void,
) {
  const form = useForm<RootKeyFormValues>({
    resolver: zodResolver(rootKeySchema),
    defaultValues,
  });
  const bodyRef = useRef<HTMLDivElement>(null);

  const scrollToTop = () => {
    bodyRef.current?.scrollTo({
      top: 0,
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
    });
  };

  return { form, bodyRef, submit: form.handleSubmit(onValid, scrollToTop) };
}
