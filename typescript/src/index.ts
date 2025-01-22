import {
  configureOAuth,
  resolveFromIdentity,
  createAuthorizationUrl,
} from "@atcute/oauth-browser-client";

configureOAuth({
  metadata: {
    client_id:
      "https://bs-feeder-staging.up.railway.app/public/client-metadata.json",
    redirect_uri: "https://bs-feeder-staging.up.railway.app/oauth/callback",
  },
});
function hello() {
  console.log("hello from ts");
}

function setupButton() {
  const btn = document.querySelector("#mybtn");
  if (!btn) {
    console.log("not found");
    return;
  }

  let inputData: HTMLInputElement = document.getElementById(
    "uri-input",
  ) as HTMLInputElement;

  let val: string;

  btn.addEventListener("click", (_event: Event) => {
    console.log("hello from btns");

    val = inputData.value;
    if (!val) {
      return;
    }

    const wrap = async () => {
      const { identity, metadata } = await resolveFromIdentity(val);
      console.log(identity);
      console.log(metadata);

      // passing `identity` is optional,
      // it allows for the login form to be autofilled with the user's handle or DID
      const authUrl = await createAuthorizationUrl({
        metadata: metadata,
        identity: identity,
        scope: "atproto transition:generic transition:chat.bsky",
      });

      console.log("hello");
      console.log(authUrl);

      // redirect the user to sign in and authorize the app
      window.location.assign(authUrl);

      // if this is on an async function, ideally the function should never ever resolve.
      // the only way it should resolve at this point is if the user aborted the authorization
      // by returning back to this page (thanks to back-forward page caching)
      await new Promise((_resolve, reject) => {
        const listener = () => {
          reject(new Error(`user aborted the login request`));
        };

        window.addEventListener("pageshow", listener, { once: true });
      });
    };

    wrap();
  });
}

setupButton();
hello();
