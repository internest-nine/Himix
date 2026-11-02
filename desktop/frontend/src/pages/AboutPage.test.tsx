// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { AboutPage } from "./AboutPage";
import { productInfo } from "../product";

const { checkMock, downloadMock, progressMock } = vi.hoisted(() => ({
  checkMock: vi.fn(),
  downloadMock: vi.fn(),
  progressMock: vi.fn(),
}));

vi.mock("../platform/services", () => ({
  appServices: {
    updater: {
      check: checkMock,
      download: downloadMock,
      installAndQuit: vi.fn(),
      progress: progressMock,
    },
  },
}));

vi.mock("../i18n/i18n", () => ({
  useI18n: () => ({
    locale: "en",
    t: (key: string) => ({
      about_intro: "intro",
      about_sponsorship_title: "Sponsorship",
      settings_sponsorship_text: "text",
      about_wechat: "WeChat",
      about_alipay: "Alipay",
    })[key] ?? key,
  }),
}));

beforeEach(() => {
  checkMock.mockReset();
  downloadMock.mockReset();
  progressMock.mockReset();
});

afterEach(cleanup);

it("shows the fork name and version instead of the upstream one", () => {
  render(<AboutPage />);
  expect(screen.getByRole("heading", { name: productInfo.name })).toBeTruthy();
  expect(screen.getByText(new RegExp(`v${productInfo.version.replace(/\./g, "\\.")}`))).toBeTruthy();
});

// The offline promise now hinges on this: the page may offer an update check,
// but nothing may run until the user asks for it. A regression here would make
// the app phone home on every visit to the About page.
it("does not contact the update server on mount", () => {
  render(<AboutPage />);
  expect(checkMock).not.toHaveBeenCalled();
});

it("checks for updates only after the user presses the button", async () => {
  checkMock.mockResolvedValue({
    current_version: productInfo.version,
    available: false,
    release: null,
  });
  render(<AboutPage />);

  fireEvent.click(screen.getByRole("button", { name: /check for updates/i }));

  await waitFor(() => expect(checkMock).toHaveBeenCalledTimes(1));
  await waitFor(() => expect(screen.getByText(/latest version/i)).toBeTruthy());
});

// The tip codes are plain files under public/support and get embedded into the
// binary at build time. Renaming one, or forgetting to copy it in, renders two
// broken images with no error anywhere - so pin both paths.
it("shows both tip codes from public/support", () => {
  render(<AboutPage />);
  expect(screen.getByRole("img", { name: /wechat tip code/i }).getAttribute("src")).toBe(
    "/support/qr-wechat.png",
  );
  expect(screen.getByRole("img", { name: /alipay tip code/i }).getAttribute("src")).toBe(
    "/support/qr-alipay.png",
  );
});
