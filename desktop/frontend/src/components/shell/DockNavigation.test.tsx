// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { DockNavigation } from "./DockNavigation";

vi.mock("../../i18n/i18n", () => ({ useI18n: () => ({ locale: "en", t: (key: string) => key }) }));
beforeEach(() => vi.stubGlobal("ResizeObserver", class { observe() {} disconnect() {} }));
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

it("renders every navigation destination as a labelled dock entry and no AI entry", () => {
  render(<DockNavigation page="settings" onPageChange={vi.fn()} />);
  for (const name of ["nav_home", "nav_routing", "nav_tools", "Links", "Tools", "nav_settings", "nav_about"]) {
    const button = screen.getByRole("button", { name });
    expect(button).toBeTruthy();
    // The dock shows the written label, so the destination is readable without
    // hovering for a tooltip.
    expect(button.className).toContain("dock-item");
    expect(button.textContent).not.toBe("");
  }
  expect(screen.queryByRole("button", { name: "AI assistant" })).toBeNull();
});

it("marks the current destination", () => {
  render(<DockNavigation page="routing" onPageChange={vi.fn()} />);
  const routing = screen.getByRole("button", { name: "nav_routing" });
  expect(routing.getAttribute("aria-current")).toBe("page");
  expect(screen.getByRole("button", { name: "nav_home" }).getAttribute("aria-current")).toBeNull();
  // Blocked domains is a child screen of settings, so settings stays highlighted.
  cleanup();
  render(<DockNavigation page="blocked-domains" onPageChange={vi.fn()} />);
  expect(screen.getByRole("button", { name: "nav_settings" }).getAttribute("aria-current")).toBe("page");
});

it("moves focus with left and right arrow keys between dock entries", () => {
  render(<DockNavigation page="routing" onPageChange={vi.fn()} />);
  const routing = screen.getByRole("button", { name: "nav_routing" });
  routing.focus();
  fireEvent.keyDown(routing, { key: "ArrowRight" });
  expect(document.activeElement).toBe(screen.getByRole("button", { name: "nav_tools" }));
  fireEvent.keyDown(screen.getByRole("button", { name: "nav_tools" }), { key: "ArrowLeft" });
  expect(document.activeElement).toBe(routing);
});

it("reports the destination the user clicks", () => {
  const onPageChange = vi.fn();
  render(<DockNavigation page="home" onPageChange={onPageChange} />);
  fireEvent.click(screen.getByRole("button", { name: "Links" }));
  expect(onPageChange).toHaveBeenCalledWith("connections");
});
