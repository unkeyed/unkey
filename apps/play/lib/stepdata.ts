export interface DataEntity {
  id: number;
}
export interface Messages extends DataEntity {
  content: string;
  color: string;
}
[];
export interface Header extends DataEntity {
  header: string;
}
export interface CurlCommand extends DataEntity {
  curlCommand: string;
}

export type StepDataMap = {
  header: Header;
  messages: Messages;
  curlCommand: CurlCommand;
};

export class StepDataStore {}
