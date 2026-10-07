import { AuthorizeDeviceLogin } from "./authorize-device-login";

type PageProps = {
  searchParams: Promise<{ user_code?: string | string[] }>;
};

export default async function CLIDevicePage({ searchParams }: PageProps) {
  const params = await searchParams;
  const raw = params.user_code;
  const userCode = Array.isArray(raw) ? (raw[0] ?? "") : (raw ?? "");
  return <AuthorizeDeviceLogin userCode={userCode} />;
}
