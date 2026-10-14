import { SettingsGroups } from "@unkey/ui";
import { EnvironmentsTable } from "./environments-table";

export default function EnvironmentsSettingsPage() {
  return (
    <SettingsGroups>
      <EnvironmentsTable />
    </SettingsGroups>
  );
}
