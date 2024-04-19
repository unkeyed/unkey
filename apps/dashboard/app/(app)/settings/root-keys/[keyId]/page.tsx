import { redirect } from "next/navigation";

type Props = {
  params: {
    keyId: string;
  };
};

export default function Redirect(props: Props) {
  return redirect(`/settings/root-keys/${props.params.keyId}/permissions`);
}
