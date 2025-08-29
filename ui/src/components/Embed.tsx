// SPDX-License-Identifier: MPL-2.0

import { Dashboard } from './dashboard'
import { useCallback, useEffect, useState, useRef } from "react";
import { parseJwt, VarsParamSchema, cx, focusRing } from "../lib/utils";
import { DarkModeProvider } from "./providers/DarkModeProvider";
import { RiEyeLine, RiEyeOffLine } from "@remixicon/react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "./tremor/Dialog";
import { Button } from "./tremor/Button";
import { translate } from "../lib/translate";

export type EmbedProps = {
  baseUrl?: string;
  dashboardId: string;
  getJwt?: () => Promise<string>;
  vars?: VarsParamSchema;
  onVarsChanged?: (newVars: VarsParamSchema) => void;
  onTitleChanged?: (title: string) => void;
}

const LOCALSTORAGE_DASHBOARD_PASSWORD_PREFIX = 'shaper-dashboard-password-';

const getPublicJwt = async (baseUrl: string, dashboardId: string, password?: string): Promise<string | null> => {
  return fetch(`${baseUrl}api/auth/public`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      dashboardId,
      ...(password && { password }),
    }),
  }).then(async (response) => {
    if (response.status !== 200) {
      return null;
    }
    const res = await response.json();
    return res.jwt;
  });
}

const getVisibility = async (baseUrl: string, dashboardId: string): Promise<string> => {
  return fetch(`${baseUrl}api/public/${dashboardId}/status`, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
  }).then(async (response) => {
    if (response.status !== 200) {
      return 'private';
    }
    const res = await response.json();
    return res.visibility ?? 'private';
  });
}

const PasswordDialog = ({
  open,
  onSubmit,
  error,
}: {
  open: boolean;
  onSubmit: (password: string) => void;
  error?: string;
}) => {
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(password);
  };

  return (
    <Dialog open={open}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{translate("Password Required")}</DialogTitle>
          <DialogDescription>
            {translate("This dashboard is password protected. Please enter the password to continue.")}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="relative">
            <input
              type={showPassword ? "text" : "password"}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={translate("Enter password")}
              className={cx(
                "w-full px-3 py-2 border rounded-md pr-12",
                "bg-cbgs dark:bg-dbgs border-cb dark:border-db",
                focusRing
              )}
              minLength={1}
              required
              autoFocus
            />
            <div className="absolute right-1 top-1">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setShowPassword(!showPassword)}
                className="p-1.5"
              >
                {showPassword ? <RiEyeOffLine className="size-4" /> : <RiEyeLine className="size-4" />}
              </Button>
            </div>
          </div>
          {error && (
            <div className="text-sm text-cerr dark:text-derr">
              {error}
            </div>
          )}
        </form>

        <DialogFooter>
          <Button
            onClick={handleSubmit}
            variant="primary"
            disabled={!password.trim()}
          >
            {translate("Access Dashboard")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export function EmbedComponent({
  initialProps,
  updateSubscriber,
}: {
  initialProps: EmbedProps;
  updateSubscriber: (updateFn: (props: Partial<EmbedProps>) => void) => void;
}) {
  const [props, setProps] = useState<EmbedProps>(initialProps);
  const { onVarsChanged, onTitleChanged, getJwt } = props;
  const jwtRef = useRef<string | null>(null);
  const [showPasswordDialog, setShowPasswordDialog] = useState(false);
  const [passwordError, setPasswordError] = useState<string>("");
  const waitingForJwtCallbackRef = useRef<((jwt: string) => void) | null>(null);

  let baseUrl = props.baseUrl ?? window.shaper.defaultBaseUrl;
  if (!baseUrl.startsWith('http://') && !baseUrl.startsWith('https://') && baseUrl[0] !== "/") {
    baseUrl = "/" + baseUrl;
  }
  if (baseUrl[baseUrl.length - 1] !== "/") {
    baseUrl = baseUrl + "/";
  }

  useEffect(() => {
    updateSubscriber((newProps: Partial<EmbedProps>) => {
      setProps(prevProps => ({ ...prevProps, ...newProps }));
    });
  }, [updateSubscriber]);

  const waitForPassword = useCallback(() => {
    setShowPasswordDialog(true);
    return new Promise<string>((resolve) => {
      waitingForJwtCallbackRef.current = resolve;
    });
  }, [])

  const handleVarsChanged = useCallback((vars: VarsParamSchema) => {
    setProps(prevProps => ({ ...prevProps, vars }));
    if (onVarsChanged) {
      onVarsChanged(vars);
    }
  }, [onVarsChanged]);

  const handleDataChanged = useCallback(({ name }: { name: string }) => {
    if (onTitleChanged) {
      onTitleChanged(name);
    }
  }, [onTitleChanged]);


  const handlePasswordSubmit = async (password: string) => {
    // Try to get JWT with the password - verification happens server-side
    const jwt = await getPublicJwt(baseUrl, props.dashboardId, password);
    if (jwt) {
      localStorage.setItem(`${LOCALSTORAGE_DASHBOARD_PASSWORD_PREFIX}${props.dashboardId}`, password);
      jwtRef.current = jwt;
      setShowPasswordDialog(false);
      setPasswordError("");
      const resolveJwt = waitingForJwtCallbackRef.current;
      if (resolveJwt) {
        resolveJwt(jwt);
      }
    } else {
      setPasswordError(translate("Invalid password. Please try again."));
    }
  };

  const handleGetJwt = useCallback(async () => {
    if (jwtRef.current != null) {
      const claims = parseJwt(jwtRef.current);
      // Check if the JWT is still valid for at least 10 seconds
      if ((Date.now() / 1000) + 10 < claims.exp) {
        return jwtRef.current;
      }
    }
    if (!getJwt) {
      const visibility = await getVisibility(baseUrl, props.dashboardId);
      if (visibility === 'private') {
        throw new Error(translate("Dashboard is not public"));
      }
      let password: string | undefined = undefined;
      if (visibility === 'password-protected') {
        // Check if we have a cached password first
        const cachedPassword = localStorage.getItem(`${LOCALSTORAGE_DASHBOARD_PASSWORD_PREFIX}${props.dashboardId}`);
        if (cachedPassword) {
          // Try with cached password first
          const jwt = await getPublicJwt(baseUrl, props.dashboardId, cachedPassword);
          if (jwt) {
            password = cachedPassword;
            return jwt;
          } else {
            // Cached password is invalid, remove it and prompt for new one
            localStorage.removeItem(`${LOCALSTORAGE_DASHBOARD_PASSWORD_PREFIX}${props.dashboardId}`);
          }
        }
        if (!password) {
          await waitForPassword();
          const jwt = jwtRef.current;
          if (!jwt) {
            throw new Error(translate("Failed to get JWT for password-protected dashboard"))
          }
          return jwt
        }
      }
      const newJwt = await getPublicJwt(baseUrl, props.dashboardId, password);
      if (newJwt == null) {
        throw new Error(translate("Failed to retrieve JWT for public dashboard"));
      }
      jwtRef.current = newJwt
      return newJwt;
    }
    const newJwt = await getJwt();
    jwtRef.current = newJwt;
    return newJwt;
  }, [baseUrl, getJwt, props.dashboardId, waitForPassword]);

  return (
    <DarkModeProvider>
      <Dashboard
        id={props.dashboardId}
        baseUrl={baseUrl}
        vars={props.vars}
        getJwt={handleGetJwt}
        onVarsChanged={handleVarsChanged}
        onDataChange={handleDataChanged}
      />
      <PasswordDialog
        open={showPasswordDialog}
        onSubmit={handlePasswordSubmit}
        error={passwordError}
      />
    </DarkModeProvider>
  );
}
