"use client";

import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { SentinelPoliciesContent } from "./sentinel-policies-content";

export default function SentinelPoliciesPage() {
  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <SentinelPoliciesContent />
    </ProjectContentWrapper>
  );
}
