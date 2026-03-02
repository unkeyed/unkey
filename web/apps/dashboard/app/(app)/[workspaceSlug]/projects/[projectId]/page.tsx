import { redirect } from "next/navigation";

export default async function ProjectDetails(props: {
  params: Promise<{ workspaceSlug: string; projectId: string }>;
}) {
  const { workspaceSlug, projectId } = await props.params;
  redirect(`/${workspaceSlug}/projects/${projectId}/deployments`);
}
