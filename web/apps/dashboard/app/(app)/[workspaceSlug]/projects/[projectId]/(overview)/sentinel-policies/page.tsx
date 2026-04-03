"use client";

import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { SentinelPoliciesContent } from "./sentinel-policies-content";

// Decide if I wanna handle environments differently instead of dropdown
// Figure out a good way to render different type of forms depending on policy type. We'll have more in the future
// Maybe do a grouping similar to env keys for environments otherwise display prod policies first
export default function SentinelPoliciesPage() {
  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <SentinelPoliciesContent />
    </ProjectContentWrapper>
  );
}
