import * as Sentry from "@sentry/nextjs";
import type { NextPageContext } from "next";
import NextError, { type ErrorProps } from "next/error";

const CustomErrorComponent = (props: ErrorProps) => {
  return <NextError statusCode={props.statusCode} />;
};

CustomErrorComponent.getInitialProps = async (contextData: NextPageContext) => {
  // In case this is running in a serverless function, await this in order to give Sentry
  // time to send the error before the lambda exits
  await Sentry.captureUnderscoreErrorException(contextData);

  // This will contain the status code of the response
  return NextError.getInitialProps(contextData);
};
// biome-ignore lint/style/noDefaultExport: This is global error component and isn't actually used anywhere but is required for next.js
export default CustomErrorComponent;
