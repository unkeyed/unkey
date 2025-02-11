"use client";

import { RatelimitOverviewLogsTable } from "./table/logs-table";

export const LogsClient = () => {
  // const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  //
  // const handleDistanceToTop = useCallback((distanceToTop: number) => {
  //   setTableDistanceToTop(distanceToTop);
  // }, []);

  return <RatelimitOverviewLogsTable />;
};
