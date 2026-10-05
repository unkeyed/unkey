"use client";

import { useParams } from "next/navigation";
import { NewAppFlow } from "./new-app-flow";

export default function NewAppPage() {
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  return <NewAppFlow projectId={projectId} />;
}
