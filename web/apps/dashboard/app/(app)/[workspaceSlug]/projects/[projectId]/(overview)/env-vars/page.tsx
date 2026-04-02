"use client";

import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { DeploymentEnvVars } from "./deployment-env-vars";

export default function EnvVarsPage() {
  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <DeploymentEnvVars />
    </ProjectContentWrapper>
  );
}
