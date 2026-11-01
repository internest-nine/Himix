// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { LanguageProvider } from "../../i18n/i18n";
import { LinkFailoverBanner } from "./LinkFailoverBanner";

const renderBanner = (state: Parameters<typeof LinkFailoverBanner>[0]["state"]) =>
  render(<LanguageProvider><LinkFailoverBanner state={state} /></LanguageProvider>);

afterEach(cleanup);

describe("LinkFailoverBanner", () => {
  it("renders nothing while every link is up", () => {
    const { container } = renderBanner({ degraded: false });
    expect(container.textContent).toBe("");
  });

  it("names the dropped adapter and the one that took over", () => {
    renderBanner({
      degraded: true,
      lost_adapters: ["Wi-Fi"],
      active_adapters: ["以太网"],
    });
    expect(screen.getByRole("status").textContent).toContain("Wi-Fi 已断开");
    expect(screen.getByRole("status").textContent).toContain("流量已转到 以太网");
  });

  it("says so when nothing is left to route over", () => {
    renderBanner({ degraded: true, lost_adapters: ["Wi-Fi", "以太网"], active_adapters: [] });
    const banner = screen.getByRole("status");
    expect(banner.className).toContain("is-offline");
    expect(banner.textContent).toContain("网络恢复后会自动接上");
  });

  it("counts down before switching back", () => {
    renderBanner({
      degraded: true,
      lost_adapters: ["Wi-Fi"],
      active_adapters: ["以太网"],
      recovering: true,
      recover_in_ms: 4200,
    });
    const banner = screen.getByRole("status");
    expect(banner.className).toContain("is-recovering");
    expect(banner.textContent).toContain("已恢复");
    // The countdown is what tells the user the switch back is deliberate, not a
    // second outage.
    expect(banner.textContent).toContain("5 秒");
  });
});
