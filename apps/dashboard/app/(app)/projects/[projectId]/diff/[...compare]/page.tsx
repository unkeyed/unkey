"use client";

// import { getAuth } from "@/lib/auth";
import { sampleDiffData } from "./constants";
import { DiffViewer } from "./components/client";

interface Props {
  params: {
    projectId: string;
    compare: string[]; // [from, to] or [from, to, additional-params]
  };
  searchParams: {
    [key: string]: string | string[] | undefined;
  };
}

export default function DiffPage({ params, searchParams }: Props) {
  const [fromVersion, toVersion] = params.compare;
  
  if (!fromVersion || !toVersion) {
    return <div>Invalid comparison parameters</div>;
  }

  // TODO: make trpc query for the diff using the versions 

  return (
    <div>
      <h1>API Spec Diff: {fromVersion} → {toVersion}</h1>
      <DiffViewer diffData={sampleDiffData}/>
    </div>
  );
}