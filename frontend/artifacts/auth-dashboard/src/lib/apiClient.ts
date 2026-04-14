import { setAuthTokenGetter } from "@workspace/api-client-react";
import { useAuthStore } from "../store/authStore";

export function initApiClient() {
  setAuthTokenGetter(() => {
    return useAuthStore.getState().accessToken;
  });
}
