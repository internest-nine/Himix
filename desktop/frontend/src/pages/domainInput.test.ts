import { describe, expect, it } from "vitest";
import { detectMatchKind, extractHost, normalizeDomain, normalizeIp, normalizeMatchValue, normalizeProcess } from "./domainInput";

describe("extractHost", () => {
  it("strips scheme, path, query and port", () => {
    expect(extractHost("https://www.example.com/item/123?id=5#top")).toBe("www.example.com");
    expect(extractHost("http://example.com:8080/a/b")).toBe("example.com");
    expect(extractHost("ftp://files.example.com/pub")).toBe("files.example.com");
  });

  it("keeps a bare host untouched and lowercases it", () => {
    expect(extractHost("Example.COM")).toBe("example.com");
    expect(extractHost("  example.com  ")).toBe("example.com");
  });

  it("drops userinfo and a trailing dot", () => {
    expect(extractHost("https://user:pass@example.com/x")).toBe("example.com");
    expect(extractHost("example.com.")).toBe("example.com");
  });

  it("returns null for empty input", () => {
    expect(extractHost("")).toBeNull();
    expect(extractHost("   ")).toBeNull();
  });
});

describe("normalizeDomain", () => {
  it("reduces a full URL to its registrable host", () => {
    expect(normalizeDomain("https://www.example.com/item/123?id=5")).toBe("example.com");
    expect(normalizeDomain("https://github.com/user/repo/issues/1")).toBe("github.com");
  });

  it("drops the leading www so the rule covers every subdomain", () => {
    expect(normalizeDomain("www.example.com")).toBe("example.com");
    expect(normalizeDomain("http://WWW.Example.com/path")).toBe("example.com");
  });

  it("keeps a meaningful subdomain the user typed", () => {
    expect(normalizeDomain("m.example.com")).toBe("m.example.com");
    expect(normalizeDomain("https://api.example.com/v1/users")).toBe("api.example.com");
  });

  it("accepts a bare domain unchanged", () => {
    expect(normalizeDomain("example.com")).toBe("example.com");
    expect(normalizeDomain("corp.local")).toBe("corp.local");
  });

  it("converts internationalised domains to punycode", () => {
    expect(normalizeDomain("https://例子.测试/路径")).toBe("xn--fsqu00a.xn--0zwm56d");
  });

  it("rejects values that are not hostnames", () => {
    expect(normalizeDomain("not a domain")).toBeNull();
    expect(normalizeDomain("https://localhost:3000/")).toBeNull();
    expect(normalizeDomain("///")).toBeNull();
  });
});

describe("normalizeIp", () => {
  it("strips scheme and port but keeps a CIDR prefix", () => {
    expect(normalizeIp("http://192.168.1.10:8080/admin")).toBe("192.168.1.10");
    expect(normalizeIp("10.0.0.0/8")).toBe("10.0.0.0/8");
    expect(normalizeIp("https://10.0.0.0/16")).toBe("10.0.0.0/16");
  });

  it("treats a non-numeric slash segment as a path", () => {
    expect(normalizeIp("172.16.5.4/path")).toBe("172.16.5.4");
  });
});

describe("normalizeMatchValue", () => {
  it("only trims process names", () => {
    expect(normalizeMatchValue("  cs2.exe  ", "process")).toBe("cs2.exe");
  });

  it("routes by match type", () => {
    expect(normalizeMatchValue("https://www.example.com/a", "domain")).toBe("example.com");
    expect(normalizeMatchValue("https://1.1.1.1/x", "ip")).toBe("1.1.1.1");
  });
});

describe("normalizeProcess", () => {
  it("reduces a pasted path to the executable name", () => {
    expect(normalizeProcess("C:\\Program Files\\Steam\\steam.exe")).toBe("steam.exe");
    expect(normalizeProcess("/usr/bin/curl")).toBe("curl");
    expect(normalizeProcess("C:/Games/CS2/cs2.exe")).toBe("cs2.exe");
  });

  it("keeps a plain process name untouched", () => {
    expect(normalizeProcess("  cs2.exe  ")).toBe("cs2.exe");
    expect(normalizeProcess("chrome")).toBe("chrome");
  });

  it("rejects values the engine would refuse anyway", () => {
    expect(normalizeProcess("")).toBeNull();
    expect(normalizeProcess("   ")).toBeNull();
    expect(normalizeProcess("a:b")).toBeNull();
  });
});

describe("detectMatchKind", () => {
  it("routes a pasted URL to the domain tab", () => {
    expect(detectMatchKind("https://www.example.com/item/123?id=5")).toBe("domain");
    expect(detectMatchKind("github.com/user/repo")).toBe("domain");
    expect(detectMatchKind("example.com")).toBe("domain");
  });

  it("routes a URL pointing at a bare address to the IP tab", () => {
    // The value that produced the HM-E1301 error in the process tab.
    expect(detectMatchKind("http://203.0.113.8:8081/portal/login.jsp?fid=1")).toBe("ip");
    expect(detectMatchKind("192.168.1.1")).toBe("ip");
    expect(detectMatchKind("10.0.0.0/8")).toBe("ip");
  });

  it("routes executables and paths to the process tab", () => {
    expect(detectMatchKind("cs2.exe")).toBe("process");
    expect(detectMatchKind("chrome")).toBe("process");
    expect(detectMatchKind("C:\\Program Files\\X\\game.exe")).toBe("process");
    // A drive-qualified path contains a slash but is still a process rule.
    expect(detectMatchKind("C:/Games/CS2/cs2.exe")).toBe("process");
  });

  it("has no opinion on empty input", () => {
    expect(detectMatchKind("")).toBeNull();
    expect(detectMatchKind("   ")).toBeNull();
  });
});
