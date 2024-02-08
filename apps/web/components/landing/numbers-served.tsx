import { Container } from "@/components/landing/container";
import { FadeIn, FadeInStagger } from "@/components/landing/fade-in";
import { StatList, StatListItem } from "@/components/landing/stat-list";
import { db, schema, sql } from "@/lib/db";
import { getTotalVerifications } from "@/lib/tinybird";
import { Suspense } from "react";

export function NumbersServed() {
  return (
    <div className="mt-24 rounded-4xl sm:mt-32 lg:mt-32">
      <Container className="">
        <FadeIn className="flex items-center gap-x-8">
          <h2 className="mb-8 text-2xl font-semibold tracking-wider text-center text-black font-display sm:text-left">
            We serve
          </h2>
          <div className="flex-auto h-px" />
        </FadeIn>
        <FadeInStagger faster>
          <StatList>
            <Suspense fallback={<StatListItem label="Workspaces" value=" ∞" />}>
              <WorkspacesCounter />
            </Suspense>
            <Suspense fallback={<StatListItem label="APIs" value=" ∞" />}>
              <ApisCounter />
            </Suspense>
            <Suspense fallback={<StatListItem label="Keys" value=" ∞" />}>
              <KeysCounter />
            </Suspense>
            <Suspense fallback={<StatListItem label="Verifications" value=" ∞" />}>
              <VerificationsCounter />
            </Suspense>
          </StatList>
        </FadeInStagger>
      </Container>
    </div>
  );
}

const WorkspacesCounter: React.FC = async () => {
  const workspaces = await db
    .select({ count: sql<number>`count(*)` })
    .from(schema.workspaces)
    .then((res) => {
      const c = res.at(0)?.count;
      console.log(`found ${c} workspaces`);
      return c ?? 0;
    })
    .catch((err) => {
      console.error(err);
      return 0;
    });

  return (
    <StatListItem
      value={Intl.NumberFormat("en", { notation: "compact" }).format(workspaces)}
      label="Workspaces"
    />
  );
};

const ApisCounter: React.FC = async () => {
  const apis = await db
    .select({ count: sql<number>`count(*)` })
    .from(schema.apis)
    .then((res) => {
      const c = res.at(0)?.count;
      console.log(`found ${c} apis`);
      return c ?? 0;
    })
    .catch((err) => {
      console.error(err);
      return 0;
    });

  return (
    <StatListItem
      value={Intl.NumberFormat("en", { notation: "compact" }).format(apis)}
      label="APIs"
    />
  );
};

const KeysCounter: React.FC = async () => {
  const keys = await db
    .select({ count: sql<number>`count(*)` })
    .from(schema.keys)
    .then((res) => {
      const c = res.at(0)?.count;
      console.log(`found ${c} keys`);
      return c ?? 0;
    })
    .catch((err) => {
      console.error(err);
      return 0;
    });

  return (
    <StatListItem
      value={Intl.NumberFormat("en", { notation: "compact" }).format(keys)}
      label="Keys"
    />
  );
};

const VerificationsCounter: React.FC = async () => {
  const verifications = await getTotalVerifications({})
    .then((res) => {
      const c = res.data.reduce((acc, curr) => acc + curr.verifications, 0);
      console.log(`found ${c} verifications`);
      return c;
    })
    .catch((err) => {
      console.error(err);
      return 0;
    });

  return (
    <StatListItem
      value={Intl.NumberFormat("en", { notation: "compact" }).format(verifications)}
      label="Verifications"
    />
  );
};
