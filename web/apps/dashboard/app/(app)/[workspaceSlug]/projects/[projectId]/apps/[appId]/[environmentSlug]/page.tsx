"use client";

import { routes } from "@/lib/navigation/routes";
import { RedirectType, redirect } from "next/navigation";
import { useAppScope } from "./environment-context";

export default function EnvironmentIndex() {
  redirect(routes.projects.apps.overview(useAppScope()), RedirectType.replace);
}
