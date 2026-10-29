import { AuthorizeDeviceLogin } from "@/app/(app)/cli/device/authorize-device-login";

type PageProps = {
  searchParams: Promise<{ user_code?: string | string[] }>;
};

export default async function OAuthConsentPage({ searchParams }: PageProps) {
  const params = await searchParams;
  const raw = params.user_code;
  const userCode = Array.isArray(raw) ? (raw[0] ?? "") : (raw ?? "");
  return <AuthorizeDeviceLogin userCode={userCode} entry="redirect" />;
}
