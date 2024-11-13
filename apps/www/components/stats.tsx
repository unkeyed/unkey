import { Container } from "@/components/container";
import { StatList, StatListItem } from "@/components/stat-list";
import { db, schema, sql } from "@/lib/db";
import { FadeInStagger } from "./fade-in";

const [workspaces, apis, keys] = await Promise.all([
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
]);

export function Stats() {
  return (
    <div className="sm:px-8 flex justify-center w-full xl:px-10">
      <div className="w-full rounded-4xl py-8 lg:pl-12 lg:py-12 border-[.75px] backdrop-filter backdrop-blur stats-border-gradient text-white max-w-[1096px]">
        <Container>
          <FadeInStagger faster>
            <StatList>
              <StatListItem value={keys} label="Keys" />
              <StatListItem value={apis} label="APIs" className="mb-8" />
              <StatListItem value={workspaces} label="Workspaces" className="mb-8" />
            </StatList>
          </FadeInStagger>
        </Container>
      </div>
    </div>
  );
}
