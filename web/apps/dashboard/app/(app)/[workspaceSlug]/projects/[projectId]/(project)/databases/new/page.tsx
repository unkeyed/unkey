"use client";

import { MotionProvider } from "@/components/motion-provider";
import { CreateDatabaseFlow } from "./create-database-flow";

export default function NewDatabasePage() {
  return (
    <MotionProvider>
      <CreateDatabaseFlow />
    </MotionProvider>
  );
}
