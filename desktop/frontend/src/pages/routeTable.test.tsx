// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, expect, it, vi } from "vitest";
import type { AppPage } from "../components/shell/navigation";
import { AppShell } from "../components/shell/AppShell";
import { isRoutedPage, renderAppPage, routedPages } from "./routeTable";

vi.mock("../i18n/i18n", () => ({
  useI18n: () => ({ locale: "en", t: (key: string) => key }),
}));
vi.mock("../components/material/useCardGlowField", () => ({ useCardGlowField: () => {} }));
vi.mock("../components/shell/TitleBar", () => ({ TitleBar: () => null }));

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

// Every id the shell can be asked to show. Home is the persistent page and is
// deliberately routed separately, but it must still be a known id.
const allPages: AppPage[] = [
  "home",
  "routing",
  "health",
  "connections",
  "tools",
  "settings",
  "blocked-domains",
  "about",
  "appearance",
];

const routeProps = {
  adapterRuntime: undefined,
  enginePhase: undefined,
  connectionsNavigation: { adapter: "", revision: 0 },
  onNavigate: () => {},
  locale: "zh",
};

it("covers every shell page exactly once", () => {
  expect([...routedPages, "home"].sort()).toEqual([...allPages].sort());
  expect(new Set(routedPages).size).toBe(routedPages.length);
});

it("renders something for every on-demand page and nothing for home", () => {
  for (const page of allPages) {
    const element = renderAppPage(page, routeProps);
    if (page === "home") {
      expect(element).toBeNull();
      continue;
    }
    expect(element, `${page} renders a blank screen`).not.toBeNull();
  }
  expect(isRoutedPage("home")).toBe(false);
  expect(isRoutedPage("settings")).toBe(true);
});

it("keeps each destination on its own component", () => {
  const types = new Set(
    (routedPages as readonly AppPage[])
      .map(page => renderAppPage(page, routeProps))
      .filter(Boolean)
      // Every branch is wrapped in Suspense for the lazy chunk, so unwrap it.
      .map(element => ((element as { props: { children: { type: unknown } } }).props.children.type)),
  );
  expect(types.size).toBe(routedPages.length);
});

// End-to-end within the shell: click the real dock entry, the real page layer
// must become the visible one. This is what a user does first in every session.
function ShellHarness() {
  const [page, setPage] = useState<AppPage>("home");
  return (
    <AppShell
      page={page}
      onPageChange={setPage}
      pageDirection="forward"
      animatePage={false}
      persistentPage="home"
      persistentChildren={<div data-testid="page-home">home</div>}
      renderPage={item => <div data-testid={`page-${item}`}>{item}</div>}
    />
  );
}

const visiblePage = () => {
  const rendered = screen.getAllByTestId(/^page-/).filter(node => !node.closest("[hidden]"));
  return rendered.map(node => node.dataset.testid?.replace("page-", ""));
};

it("shows exactly one page after clicking each dock destination", () => {
  vi.stubGlobal("ResizeObserver", class { observe() {} disconnect() {} });
  render(<ShellHarness />);
  expect(visiblePage()).toEqual(["home"]);

  const destinations: Array<[string, string]> = [
    ["nav_home", "home"],
    ["nav_routing", "routing"],
    ["nav_tools", "health"],
    ["Links", "connections"],
    ["Tools", "tools"],
    ["nav_settings", "settings"],
    ["nav_about", "about"],
  ];
  for (const [label, expected] of destinations) {
    fireEvent.click(screen.getByRole("button", { name: label }));
    expect(visiblePage(), `clicking ${label} should show ${expected}`).toEqual([expected]);
  }
});
