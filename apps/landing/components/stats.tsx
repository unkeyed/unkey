import { Container } from "@/components/container";
import { StatList, StatListItem } from "@/components/stat-list";
import { db, schema, sql } from "@/lib/db";
import { getTotalVerifications } from "@/lib/tinybird";

const [apis, keys, totalVerifications] = await Promise.all([
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

export function Stats() {
  return (
    <div className="flex justify-center w-full xl:px-10">
      <div className="w-full rounded-4xl py-8 lg:pl-12 lg:py-12 border-[.75px] backdrop-filter backdrop-blur stats-border-gradient text-white max-w-[1096px]">
        <Container>
          <StatList>
            <StatListItem value={totalVerifications} label="Verifications" />
            <StatListItem value={apis} label="APIs" className="mb-8" />
            <StatListItem value={keys} label="Keys" />
          </StatList>
        </Container>
      </div>
    </div>
  );
}
