"use client";

import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: {
    apiId: string;
  };
};

export default function SettingsPage(props: Props) {
  const { apiId } = props.params;
  return (
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/apis/${apiId}/settings`,
          text: "Settings",
        }}
      />
      <SettingsClient apiId={apiId} />
    </div>
  );
}
