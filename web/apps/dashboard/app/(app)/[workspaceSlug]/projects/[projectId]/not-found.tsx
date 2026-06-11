"use client";
import { NotFoundState } from "@/components/not-found-state";

export default function ProjectNotFound() {
  return (
    <NotFoundState description="We couldn't find the project or deployment that you're looking for." />
  );
}
