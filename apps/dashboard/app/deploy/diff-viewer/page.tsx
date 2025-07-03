"use client";

// import { getAuth } from "@/lib/auth";
import { sampleDiffData } from "./constants";
import { DiffViewer } from "./client";

export const dynamic = "force-dynamic";

export default function DiffViewerPage () {

  return <DiffViewer diffData={sampleDiffData}/>

}