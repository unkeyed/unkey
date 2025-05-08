"use client";

import { KeysDetailsLogsControlCloud } from "./components/control-cloud";
import { KeysDetailsLogsControls } from "./components/controls";
import { KeyDetailsLogsTable } from "./components/table/logs-table";

export const KeyDetailsLogsClient = ({
  apiId,
  keyspaceId,
  keyId,
}: {
  keyId: string;
  apiId: string;
  keyspaceId: string;
}) => {
  // const [selectedLog, setSelectedLog] = useState<KeysOverviewLog | null>(null);
  // const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  //
  // const handleDistanceToTop = useCallback((distanceToTop: number) => {
  //   setTableDistanceToTop(distanceToTop);
  // }, []);
  //
  // const handleSelectedLog = useCallback((log: KeysOverviewLog | null) => {
  //   setSelectedLog(log);
  // }, []);
  //
  return (
    <div className="flex flex-col">
      <KeysDetailsLogsControls apiId={apiId} />
      <KeysDetailsLogsControlCloud />
      <div className="flex flex-col">
        {/* <KeysOverviewLogsCharts apiId={apiId} onMount={handleDistanceToTop} /> */}
        <KeyDetailsLogsTable apiId={apiId} keyspaceId={keyspaceId} keyId={keyId} />
      </div>
      {/* <KeysOverviewLogDetails */}
      {/*   apiId={apiId} */}
      {/*   distanceToTop={tableDistanceToTop} */}
      {/*   setSelectedLog={handleSelectedLog} */}
      {/*   log={selectedLog} */}
      {/* /> */}
    </div>
  );
};
