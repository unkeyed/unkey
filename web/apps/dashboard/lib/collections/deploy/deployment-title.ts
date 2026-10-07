import { imageRefDisplay } from "@/lib/docker-image-ref";
import { shortenId } from "@/lib/shorten-id";

type TitledDeployment = {
  id: string;
  source: "git" | "oci" | "unknown";
  gitCommitMessage: string | null;
  requestedImage: string | null;
  resolvedImage: string | null;
};

export function deploymentTitle(deployment: TitledDeployment): string {
  if (deployment.gitCommitMessage) {
    return deployment.gitCommitMessage;
  }
  const image = deployment.requestedImage ?? deployment.resolvedImage;
  if (deployment.source === "oci" && image) {
    return imageRefDisplay(image);
  }
  return shortenId(deployment.id);
}
