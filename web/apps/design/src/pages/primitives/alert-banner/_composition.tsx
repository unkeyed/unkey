import { TriangleWarning2 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
} from "@unkey/ui";
import type { ReactNode } from "react";

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex w-full flex-col gap-1.5">
      <span className="font-mono text-gray-9 text-xs">{label}</span>
      {children}
    </div>
  );
}

const DESCRIPTION = "The build could not resolve the git branch for this project.";
const TITLE = "Deployment failed";

export function AlertBannerComposition() {
  return (
    <div className="flex w-full flex-col gap-6">
      <Row label="description">
        <AlertBanner variant="error">
          <AlertBannerDescription>{DESCRIPTION}</AlertBannerDescription>
        </AlertBanner>
      </Row>

      <Row label="icon + description">
        <AlertBanner variant="error">
          <TriangleWarning2 iconSize="md-regular" />
          <AlertBannerDescription>{DESCRIPTION}</AlertBannerDescription>
        </AlertBanner>
      </Row>

      <Row label="icon + description + actions">
        <AlertBanner variant="error">
          <TriangleWarning2 iconSize="md-regular" />
          <AlertBannerDescription>{DESCRIPTION}</AlertBannerDescription>
          <AlertBannerActions>
            <Button variant="outline" size="md" className="bg-background">
              Redeploy
            </Button>
          </AlertBannerActions>
        </AlertBanner>
      </Row>

      <Row label="title + description">
        <AlertBanner variant="error">
          <AlertBannerTitle>{TITLE}</AlertBannerTitle>
          <AlertBannerDescription>{DESCRIPTION}</AlertBannerDescription>
        </AlertBanner>
      </Row>

      <Row label="icon + title + description + actions">
        <AlertBanner variant="error">
          <TriangleWarning2 iconSize="md-regular" />
          <AlertBannerTitle>{TITLE}</AlertBannerTitle>
          <AlertBannerDescription>{DESCRIPTION}</AlertBannerDescription>
          <AlertBannerActions>
            <Button variant="outline" size="md" className="bg-background">
              Redeploy
            </Button>
          </AlertBannerActions>
        </AlertBanner>
      </Row>

      <Row label="title + actions">
        <AlertBanner variant="error">
          <AlertBannerTitle>{TITLE}</AlertBannerTitle>
          <AlertBannerActions>
            <Button variant="outline" size="md" className="bg-background">
              Redeploy
            </Button>
          </AlertBannerActions>
        </AlertBanner>
      </Row>

      <Row label="wrapping description, icon stays centred">
        <AlertBanner variant="warning">
          <TriangleWarning2 iconSize="md-regular" />
          <AlertBannerTitle>Compute is paused</AlertBannerTitle>
          <AlertBannerDescription>
            This workspace has no active Compute plan, so existing projects stay visible but you
            cannot create or deploy. Choose a plan to start deploying again, or remove the projects
            you no longer need.
          </AlertBannerDescription>
          <AlertBannerActions>
            <Button variant="outline" size="md" className="bg-background">
              Choose a plan
            </Button>
          </AlertBannerActions>
        </AlertBanner>
      </Row>
    </div>
  );
}
