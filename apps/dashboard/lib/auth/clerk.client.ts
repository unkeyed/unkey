"use client";
import { useOrganizationList, useUser } from "@clerk/nextjs";
import type { ClientAuth } from "./interface.client";

export class ClerkClient implements ClientAuth {
  useUser() {
    return useUser();
  }
  useOrganizationList() {
    const { setActive } = useOrganizationList();
    return {
      setActive,
    };
  }
}
