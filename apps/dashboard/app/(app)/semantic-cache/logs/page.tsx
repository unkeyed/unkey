// import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
// import {
//   Dialog,
//   DialogContent,
//   DialogDescription,
//   DialogFooter,
//   DialogHeader,
//   DialogTitle,
//   DialogTrigger,
// } from "@/components/ui/dialog";
// import { Input } from "@/components/ui/input";
// import { Label } from "@/components/ui/label";
// import { Separator } from "@/components/ui/separator";

// import {
//   Table,
//   TableBody,
//   TableCaption,
//   TableCell,
//   TableHead,
//   TableHeader,
//   TableRow,
// } from "@/components/ui/table";

import { getAllSemanticCacheLogs } from "@/lib/tinybird";
import { IntervalSelect } from "../../apis/[apiId]/select";
import Table from "./table";

// const formatDate = (timestamp: string | number | Date): string => {
//   const date = new Date(timestamp);
//   const options: Intl.DateTimeFormatOptions = {
//     month: "long",
//     day: "numeric",
//     hour: "2-digit",
//     minute: "2-digit",
//     second: "2-digit",
//   };
//   return date.toLocaleDateString("en-US", options);
// };

export default async function SemanticCacheLogsPage() {
  const { data } = await getAllSemanticCacheLogs({ limit: 1000 });

  return <Table data={data} />;
}
