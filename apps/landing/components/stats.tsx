import { Container } from "@/components/container";
import { StatList, StatListItem } from "@/components/stat-list";
import { db, schema, sql } from "@/lib/db";
import { getTotalVerifications } from "@/lib/tinybird";
import { cn } from "@/lib/utils";
import { motion } from "framer-motion";
import { NumberTicker } from "./number-ticker";

const [workspaces, apis, keys, totalVerifications] = await Promise.all([
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.workspaces)
    .then((res) => res.at(0)?.count ?? 0)
    .catch((err) => {
      console.error(err);
      return 0;
    }),
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.apis)
    .then((res) => res.at(0)?.count ?? 0)
    .catch((err) => {
      console.error(err);
      return 0;
    }),
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.keys)
    .then((res) => res.at(0)?.count ?? 0)
    .catch((err) => {
      console.error(err);
      return 0;
    }),
  getTotalVerifications({})
    .then((res) => {
      return res.data.reduce((acc, curr) => acc + curr.verifications, 0);
    })
    .catch((err) => {
      console.error(err);
      return 0;
    }),
  { next: { revalidate: 3600 } },
]);

function StatsItem({
  value,
  label,
  className,
}: { value: number; label: string; className?: string }) {
  return (
    <div
      className={cn(
        "flex-col-reverse pl-8 border-white/[.15] border-l max-w-[200px] md:mb-0",
        className,
      )}
    >
      <dd className="text-4xl font-semibold font-display stats-number-gradient">
        <NumberTicker value={value} />
      </dd>
      <dt className="mt-2 text-white/50">{label}</dt>
    </div>
  );
}

export function Stats() {
  const containerVariants = {
    hidden: { opacity: 0 },
    show: {
      opacity: 1,
      transition: {
        staggerChildren: 0.2, // Adjust the delay between each child here
      },
    },
  };
  return (
    <motion.div
      className="flex justify-center w-full xl:px-10"
      variants={containerVariants}
      initial="hidden"
      animate="show"
    >
      <div className="w-full rounded-4xl py-8 lg:pl-12 lg:py-12 border-[.75px] backdrop-filter backdrop-blur stats-border-gradient text-white max-w-[1096px]">
        <Container>
          <StatList>
            <StatsItem value={totalVerifications} label="Verifications" />
            <StatsItem value={keys} label="Keys" />
            <StatsItem value={apis} label="APIs" className="mb-8" />
            <StatsItem value={workspaces} label="Workspaces" className="mb-8" />
          </StatList>
        </Container>
      </div>
    </motion.div>
  );
}
