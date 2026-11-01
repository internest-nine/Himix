// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { AppShell } from "./AppShell";
import { usePageActive } from "./PageActivity";
import type { AppPage } from "./navigation";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("../../platform/services", () => ({ appServices: { settings: { get: mocks.get } } }));
vi.mock("../material/useCardGlowField", () => ({ useCardGlowField: () => {} }));
vi.mock("./TitleBar", () => ({ TitleBar: () => null }));
vi.mock("./DockNavigation", () => ({
  DockNavigation: ({ page, onPageChange }: { page: AppPage; onPageChange: (page: AppPage) => void }) => (
    <nav>
      <button onClick={() => onPageChange("routing")}>routing</button>
      <span data-testid="current-page">{page}</span>
    </nav>
  ),
}));
vi.mock("../../i18n/i18n", () => ({ useI18n: () => ({ locale: "en" }) }));
afterEach(cleanup);
beforeEach(() => mocks.get.mockReset().mockResolvedValue({}));

function DraftPage({ page }: { page: AppPage }) {
  const [draft, setDraft] = useState("");
  const active = usePageActive();
  return <div data-testid={page} data-active={active} style={{ overflow: "auto", height: 100 }}>
    <input aria-label={`${page} draft`} value={draft} onChange={event => setDraft(event.target.value)} />
  </div>;
}

it("mounts pages on demand and preserves drafts, scroll and DOM identity across navigation", () => {
  const shell = (page: AppPage) => <AppShell page={page} onPageChange={() => {}} pageDirection="forward" animatePage renderPage={item => <DraftPage page={item} />} />;
  const view = render(shell("routing"));
  expect(screen.queryByTestId("settings")).toBeNull();
  const draft = screen.getByRole("textbox", { name: "routing draft" });
  const scroller = screen.getByTestId("routing");
  fireEvent.change(draft, { target: { value: "unfinished.exe" } });
  scroller.scrollTop = 240;
  view.rerender(shell("settings"));
  expect(screen.queryByRole("textbox", { name: "routing draft" })).toBeNull();
  expect(scroller.dataset.active).toBe("false");
  view.rerender(shell("routing"));
  expect(screen.getByRole("textbox", { name: "routing draft" })).toBe(draft);
  expect((draft as HTMLInputElement).value).toBe("unfinished.exe");
  expect(scroller.scrollTop).toBe(240);
  expect(scroller.dataset.active).toBe("true");
  expect(screen.getByTestId("settings").dataset.active).toBe("false");
});

// The assistant surface was removed from this fork, so the shell must not read
// AI settings or mount anything AI-related any more.
it("renders the shell without touching AI settings", () => {
  render(<AppShell page="home" onPageChange={() => {}} pageDirection="forward" animatePage={false}>
    <div>home body</div>
  </AppShell>);
  expect(screen.getByText("home body")).toBeTruthy();
  expect(screen.getByTestId("current-page").textContent).toBe("home");
  expect(mocks.get).not.toHaveBeenCalled();
});
