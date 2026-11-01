import { Spinner } from "@fluentui/react-components";
import { lazy, Suspense, type ReactNode } from "react";
import type { AppPage } from "../components/shell/navigation";
import type { EnginePhase, HomeAdapter } from "../state/useEngineState";
import type { ConnectionsNavigation } from "./connectionNavigation";

const AppearanceLab = lazy(() => import("./AppearanceLab").then((module) => ({ default: module.AppearanceLab })));
const AboutPage = lazy(() => import("./AboutPage").then((module) => ({ default: module.AboutPage })));
const BlockedDomainsPage = lazy(() => import("./BlockedDomainsPage").then((module) => ({ default: module.BlockedDomainsPage })));
const HealthPage = lazy(() => import("./HealthPage").then((module) => ({ default: module.HealthPage })));
const ConnectionsPage = lazy(() => import("./ConnectionsPage").then((module) => ({ default: module.ConnectionsPage })));
const RoutingPage = lazy(() => import("./RoutingPage").then((module) => ({ default: module.RoutingPage })));
const ToolsPage = lazy(() => import("./ToolsPage").then((module) => ({ default: module.ToolsPage })));
const SettingsPage = lazy(() => import("./SettingsPage").then((module) => ({ default: module.SettingsPage })));

/** Pages the shell can mount on demand. Home is excluded on purpose: it stays
    mounted behind every other page so its engine polling never restarts. */
export const routedPages = [
  "routing",
  "health",
  "connections",
  "tools",
  "settings",
  "blocked-domains",
  "about",
  "appearance",
] as const satisfies readonly AppPage[];

export type RoutedPage = typeof routedPages[number];

export type RouteProps = {
  adapterRuntime: HomeAdapter[] | undefined;
  enginePhase: EnginePhase | undefined;
  connectionsNavigation: ConnectionsNavigation;
  onNavigate: (page: AppPage) => void;
  locale: string;
};

export function isRoutedPage(page: AppPage): page is RoutedPage {
  return (routedPages as readonly AppPage[]).includes(page);
}

export function renderAppPage(target: AppPage, props: RouteProps): ReactNode {
  const { adapterRuntime, enginePhase, connectionsNavigation, onNavigate, locale } = props;

  // Home renders through the shell's persistent layer, so nothing to mount here.
  if (!isRoutedPage(target)) return null;

  const page = (() => {
    switch (target) {
      case "appearance":
        // The lab ships in dev builds only.
        return import.meta.env.DEV ? <AppearanceLab /> : null;
      case "about":
        return <AboutPage />;
      case "tools":
        return <ToolsPage />;
      case "settings":
        return <SettingsPage adapterRuntime={adapterRuntime} onOpenBlockedDomains={() => onNavigate("blocked-domains")} />;
      case "blocked-domains":
        return <BlockedDomainsPage onBack={() => onNavigate("settings")} />;
      case "health":
        return <HealthPage adapterRuntime={adapterRuntime} enginePhase={enginePhase} />;
      case "connections":
        return (
          <ConnectionsPage
            initialAdapter={connectionsNavigation.adapter}
            adapterRevision={connectionsNavigation.revision}
            adapterRuntime={adapterRuntime ?? []}
          />
        );
      case "routing":
        return <RoutingPage />;
      default: {
        // Adding a RoutedPage without a branch here fails to compile.
        const unreachable: never = target;
        return unreachable;
      }
    }
  })();

  if (!page) return null;
  return (
    <Suspense fallback={<div className="page-loading"><Spinner size="small" label={locale === "en" ? "Loading page…" : "正在加载页面…"} /></div>}>
      {page}
    </Suspense>
  );
}
