import { redirect } from "next/navigation";

export default async function ProjectDetails(props: {
  params: { workspaceSlug: string; projectId: string };
}) {
  const { workspaceSlug, projectId } = props.params;
  redirect(`/${workspaceSlug}/projects/${projectId}/deployments`);
}
