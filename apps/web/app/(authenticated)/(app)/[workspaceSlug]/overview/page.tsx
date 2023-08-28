import { redirect } from "next/navigation";

type Props = {
  params: {
    workspaceSlug: string;
  };
};

export default function OverviewPage(props: Props) {
  return redirect(`/${props.params.workspaceSlug}/apis`);
}
