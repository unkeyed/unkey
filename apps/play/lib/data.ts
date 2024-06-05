const apiId = process.env.NEXT_PUBLIC_PLAYGROUND_API_ID;

export type Step = {
  header: string;
  messages: { content: string; color: string }[];
  curlCommand: string;
};
export type Message = { content: string; color: string };
type Steps = {
  header: string;
  messages: { content: string; color: string }[];
  curlCommand: string;
}[];

export function getStepsData() {
  const steps: Steps = [];

  // Step 1 Create your first key
  const step1Header = "";
  const step1Messages = [
    { content: "Step 1: Create your first key", color: "text-white" },
    {
      content:
        "Welcome to the Unkey playground. Here, you can test how Unkey's API works using curl commands. The first step is to create a key for a user. They will use this key to authenticate with your API. Normally, you would need an apiId and root key, but we've already set this up for you. Usually, this setup would be done in your Unkey dashboard. For now, leave the <token> in place. This is where your bearer token or root key would go. For each step, type in or copy the curl command below into the input at the bottom and press enter. Alternatively, you can just click the curl command and it will be sent off for you.",
      color: "text-white",
    },
    {
      content:
        "As you can see its a typical POST request with a url, headers and data. The Bearer in the first header authenticates you in the Unkey system. The data object passes what api you would like to create a key for.",
      color: "text-white",
    },
    {
      content: `curl --request POST --url https://api.unkey.dev/v1/keys.createKey
--header 'Authorization: Bearer <token>
--header 'Content-Type: application/json
--data '{"apiId": "${apiId}"}'`,
      color: "text-white",
    },
  ];
  const step1CurlCommand = `curl --request POST --url https://api.unkey.dev/v1/keys.createKey
--header 'Authorization: Bearer <token>
--header 'Content-Type: application/json
--data '{"apiId": "${apiId}"}'`;
  steps.push({
    header: step1Header,
    messages: step1Messages,
    curlCommand: step1CurlCommand,
  });
  // Step 2 Get the key we just created
  const step2Header = "Step 2: Get the key we just created";
  const step2Messages = [
    {
      content:
        "Nice Job! Now that we have created a user key let us use the getKey endpoint to get some info. A typical key will have useful information like roles, permissions, remaining uses, ownerId and more. A full list can be found on our docs https://www.unkey.com/docs/api-reference/keys/get",
      color: "text-white",
    },
  ];
  const step2CurlCommand = `curl --request GET
--url https://api.unkey.dev/v1/keys.getKey?keyId=<keyId>
--header 'Authorization: Bearer <token>'`;
  steps.push({
    header: step2Header,
    messages: step2Messages,
    curlCommand: step2CurlCommand,
  });
  // Step 3 Verify the key
  const step3Header = "Step 3: Verify the key";
  const step3Messages = [
    {
      content:
        "lets go! Looks like that worked and returned some data. This object can get much larger depending on the options added to the key. Now we can verify the key we just created to make sure it will work for a user. Each verification will add some analytics data we will get in a later step.",
      color: "text-white",
    },
  ];
  const step3CurlCommand = `curl --request POST
--url https://api.unkey.dev/v1/keys.verifyKey
--header 'Content-Type: application/json'
--data '{"apiId": "${apiId}", "key": "<key>"}'`;
  steps.push({
    header: step3Header,
    messages: step3Messages,
    curlCommand: step3CurlCommand,
  });
  // Step 4  Update the key with ownerId
  const step4Header = "Step 4: Update the key with ownerId";
  const step4Messages = [
    {
      content:
        "You will notice the enabled: true meaning meaning it can be used for authenticating. Now lets try to update the key with an ownerId. The ownerId can help you link your user to a spacific key or set of keys. For example if our customer was Acme we could mark all the keys with Acme_Company. This could then be searched to see all keys witht he Acme_Company ownerId. Making it easier for you to track keys for each customer. Lets add that now. Feel free to put any OwnerId you want in place of user_1234.",
      color: "text-white",
    },
  ];
  const step4CurlCommand = `curl --request POST
--url https://api.unkey.dev/v1/keys.updateKey
--header 'Authorization: Bearer <token>'
--header 'Content-Type: application/json'
--data '{"keyId": "<keyId>", "ownerId": "user_1234"}'`;
  steps.push({
    header: step4Header,
    messages: step4Messages,
    curlCommand: step4CurlCommand,
  });
  //Step 5 Verify the key again
  const step5Header = "Step 5: Verify the key again";
  const step5Messages = [
    {
      content:
        "If this worked correctly you should get back a {} in response. This is normal and signifies success. If there was an error you would have gotten back an error in stead. Lets now verify the key just to make sure your key is updated with the ownerId. Just to give you piece of mind.",
      color: "text-white",
    },
  ];
  const step5CurlCommand = `curl --request POST
--url https://api.unkey.dev/v1/keys.verifyKey
--header 'Content-Type: application/json'
--data '{"apiId": "${apiId}", "key": "<key>"}'`;
  steps.push({
    header: step5Header,
    messages: step5Messages,
    curlCommand: step5CurlCommand,
  });
  // Step 6 Update with an expiration date
  const step6Header = "Step 6: Update with an expiration date";
  const step6Messages = [
    {
      content:
        "The response from step 5 should show the ownerId you enter in step 4. If you want to only allow a user access for a day or month. This can be done by adding an expiration date in unix timestamp in milliseconds. This will disable the key from being used after that time has passed. You may also change that expiration to a later time after it expires. Say a user pays for another month it can be re activated.",
      color: "text-white",
    },
  ];
  const step6CurlCommand = `curl --request POST
--url https://api.unkey.dev/v1/keys.updateKey
--header 'Authorization: Bearer <token>'
--header 'Content-Type: application/json'
--data '{"keyId": "<keyId>", "expires": <timeStamp>}'`;
  steps.push({
    header: step6Header,
    messages: step6Messages,
    curlCommand: step6CurlCommand,
  });
  // Step 7 Verify the key again
  const step7Header = "Step 7: Verify the key again";
  const step7Messages = [
    {
      content:
        "Again we should have gotten {} in response. Lets verify the key to make sure expiration was set correctly. It will also give up more analytics data for the next step.",
      color: "text-white",
    },
  ];
  const step7CurlCommand = `curl --request POST
--url https://api.unkey.dev/v1/keys.verifyKey
--header 'Content-Type: application/json'
--data '{"apiId": "${apiId}","key": "<key>"}'`;
  steps.push({
    header: step7Header,
    messages: step7Messages,
    curlCommand: step7CurlCommand,
  });
  // Step 8
  const step8Header = "Step 8: Get analytics data";
  const step8Messages = [
    {
      content:
        "Nice! it looks like the expires value now holds the value you entered. Now lets see the analytics data on our key. This will show how many times the key was successfuly verified, ratelimited, or the usage has been exceeeded.",
      color: "text-white",
    },
  ];
  const step8CurlCommand = `curl --request GET
--url https://api.unkey.dev/v1/keys.getVerifications?keyId=<keyId>
--header 'Authorization: Bearer <token>'`;
  steps.push({
    header: step8Header,
    messages: step8Messages,
    curlCommand: step8CurlCommand,
  });
  //Step 9
  const step9Header = "Step 9: Delete the key";
  const step9Messages = [
    {
      content:
        "If it was a new key and all our verify keys worked correctly. success should show a value of 3. Lets say we no longer want this key to have any access. While we can use updateKey to set enabled to false. Here lets delete the key so it has not option of being used again. Maybe the key was leaked to a bad actor or it is no longer needed for any reason.",
      color: "text-white",
    },
  ];
  const step9CurlCommand = `curl --request POST
--url https://api.unkey.dev/v1/keys.deleteKey
--header 'Content-Type: application/json'
--data '{"keyId": "<keyId>"}'`;
  steps.push({
    header: step9Header,
    messages: step9Messages,
    curlCommand: step9CurlCommand,
  });
  //Step 10 Last Verification
  const step10Header = "Step 10: Last verification";
  const step10Messages = [
    {
      content:
        "Now lets verify that deleting a key worked as it should. Use the verifyKey route to check the key again. This should now return valid: false meaning this key is no longer a valid Unkey key.",
      color: "text-white",
    },
  ];
  const step10CurlCommand = `curl --request POST
--url https://api.unkey.dev/v1/keys.verifyKey
--header 'Content-Type: application/json'
--data '{"apiId": "${apiId}", "key": "<key>"}'`;
  steps.push({
    header: step10Header,
    messages: step10Messages,
    curlCommand: step10CurlCommand,
  });
  //Step 11 Congrats!
  const step11Header = "Congrats!";
  const step11Messages = [
    {
      content:
        "Nice job. I hope this was useful in some way on learning about some Unkey basics. To learn more visit our docs. To get started feel free to setup an unkey dashboard for your project.",
      color: "text-white",
    },
  ];
  const step11CurlCommand = "";
  steps.push({
    header: step11Header,
    messages: step11Messages,
    curlCommand: step11CurlCommand,
  });

  return steps;
}
