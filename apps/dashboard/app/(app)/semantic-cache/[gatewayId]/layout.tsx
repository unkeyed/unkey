import { Navbar } from "@/components/dashboard/navbar";

export default async function SemanticCacheLayout({
  params,
  children,
}: { params: { gatewayId: string }; children: React.ReactNode }) {
  const navigation = [
    {
      label: "Logs",
      href: `/semantic-cache/${params.gatewayId}/logs`,
      segment: "logs",
    },
    {
      label: "Analytics",
      href: `/semantic-cache/${params.gatewayId}/analytics`,
      segment: "analytics",
    },
    {
      label: "Settings",
      href: `/semantic-cache/${params.gatewayId}/settings`,
      segment: "settings",
    },
  ];

  return (
    <>
      <Navbar navigation={navigation} />
      <div>{children}</div>
    </>
  );
}
