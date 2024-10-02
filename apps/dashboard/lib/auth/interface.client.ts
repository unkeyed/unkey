export interface ClientAuth {
  useOrganizationList(): {
    setActive: ((args: any) => Promise<void>) | undefined;
  };

  useUser():
    | {
        isLoaded: false;
        isSignedIn: undefined;
        user: undefined;
      }
    | {
        isLoaded: true;
        isSignedIn: false;
        user: null;
      }
    | {
        isLoaded: true;
        isSignedIn: true;
        user: {
          id: string;
        };
      };
}
