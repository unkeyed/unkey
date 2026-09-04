export function imageTag(image: string): string {
  const lastSegment = image.slice(image.lastIndexOf("/") + 1);
  const cut = lastSegment.lastIndexOf(":");
  return cut === -1 ? "latest" : lastSegment.slice(cut + 1);
}

export function imageDisplay(image: string): string {
  const cut = image.indexOf("/");
  if (cut === -1) {
    return image;
  }
  const host = image.slice(0, cut);
  const isRegistryHost = host.includes(".") || host.includes(":") || host === "localhost";
  return isRegistryHost ? image.slice(cut + 1) : image;
}
