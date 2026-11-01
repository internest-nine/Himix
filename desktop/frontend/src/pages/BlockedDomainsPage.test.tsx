// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { BlockedDomainsPage } from "./BlockedDomainsPage";
import { PageActivity } from "../components/shell/PageActivity";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  remove: vi.fn(),
  clear: vi.fn(),
  notify: vi.fn(),
}));

vi.mock("../platform/services", () => ({
  appServices: { blockedDomains: { list: mocks.list, remove: mocks.remove, clear: mocks.clear } },
}));
vi.mock("../components/notifications/AppNotifications", () => ({
  useAppNotifications: () => ({ notify: mocks.notify }),
}));
vi.mock("../i18n/i18n", () => ({
  useI18n: () => ({ locale: "en", t: (key: string, values?: Record<string, string | number>) => (values ? `${key}:${Object.values(values).join(",")}` : key) }),
}));
vi.mock("../platform/serialPoll", () => ({ startSerialPoll: () => () => {} }));

const snapshot = {
  enabled: true,
  use_expiry: true,
  entries: [{ adapter: "Ethernet", domain: "example.com", expires_at: "", remaining_seconds: 0, permanent: true }],
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.list.mockResolvedValue(snapshot);
  mocks.remove.mockResolvedValue(undefined);
  mocks.clear.mockResolvedValue(undefined);
  class Observer { observe() {} unobserve() {} disconnect() {} }
  vi.stubGlobal("ResizeObserver", Observer);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const renderPage = (onBack = vi.fn()) => render(
  <PageActivity.Provider value>
    <BlockedDomainsPage onBack={onBack} />
  </PageActivity.Provider>,
);

it("returns to settings from the back button", async () => {
  const onBack = vi.fn();
  renderPage(onBack);
  const back = await screen.findByRole("button", { name: /返回设置|Back to settings/ });
  fireEvent.click(back);
  expect(onBack).toHaveBeenCalledOnce();
});

it("lists the recorded domains and deletes a single one", async () => {
  renderPage();
  expect(await screen.findByText("example.com")).toBeTruthy();
  await waitFor(() => expect(mocks.list).toHaveBeenCalled());

  const removeButtons = screen.getAllByRole("button").filter(button => button.getAttribute("aria-label")?.includes("example.com"));
  expect(removeButtons.length).toBe(1);
  fireEvent.click(removeButtons[0]);
  await waitFor(() => expect(mocks.remove).toHaveBeenCalledWith("Ethernet", "example.com"));
});

it("keeps the clear-all action behind its confirmation dialog", async () => {
  renderPage();
  await screen.findByText("example.com");
  const clearButton = screen.getByRole("button", { name: "blocked_clear_all" });
  expect(mocks.clear).not.toHaveBeenCalled();
  fireEvent.click(clearButton);
  const confirm = await screen.findByRole("button", { name: /Clear records|确认清空/ });
  fireEvent.click(confirm);
  await waitFor(() => expect(mocks.clear).toHaveBeenCalledOnce());
});
