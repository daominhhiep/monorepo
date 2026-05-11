import { createConnectTransport } from "@connectrpc/connect-web";
import { createClient } from "@connectrpc/connect";
import { APIService } from "../gen/apps/webapp/v1/api_pb";

// The Vite dev server proxies these calls to the BFF; in prod we serve the
// SPA from the same origin behind the BFF/ingress so this path stays valid.
const transport = createConnectTransport({
  baseUrl: window.location.origin,
  useBinaryFormat: false,
  fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
});

export const api = createClient(APIService, transport);
