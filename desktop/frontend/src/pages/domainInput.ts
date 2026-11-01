/**
 * Match-value normalisation for the routing editor.
 *
 * Users paste whatever is on their clipboard — a full URL from the address bar,
 * a host:port, a domain with a path. The routing engine only accepts a bare
 * host (or CIDR for IP rules), so the value is reduced here before validation
 * instead of rejecting the paste and asking the user to edit it by hand.
 */

const SCHEME = /^[a-z][a-z0-9+.-]*:\/\//i;

/** Strip scheme, userinfo, path, query, fragment and port. */
export function extractHost(raw: string): string | null {
  let value = raw.trim();
  if (!value) return null;
  value = value.replace(SCHEME, "");
  value = value.replace(/^[^/@]*@/, "");
  value = value.split(/[/?#]/)[0];
  value = value.replace(/:\d+$/, "");
  value = value.replace(/\.$/, "");
  return value ? value.toLowerCase() : null;
}

const HOSTNAME = /^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

function toAscii(host: string): string {
  // URL() resolves internationalised domains to punycode for us.
  try {
    return new URL(`http://${host}`).hostname;
  } catch {
    return host;
  }
}

/**
 * Reduce a pasted value to a routing domain. The leading `www.` is dropped
 * because a domain rule already covers every subdomain — keeping it would only
 * narrow the rule in a way users do not expect.
 */
export function normalizeDomain(raw: string): string | null {
  const host = extractHost(raw);
  if (!host) return null;
  const ascii = toAscii(host).replace(/^www\./, "");
  if (!ascii.includes(".")) return null;
  if (!HOSTNAME.test(ascii)) return null;
  return ascii;
}

/**
 * IP rules accept a bare address or CIDR. A trailing `/` segment is kept only
 * when it is a numeric prefix length; anything else is a URL path.
 */
export function normalizeIp(raw: string): string | null {
  let value = raw.trim().replace(SCHEME, "");
  value = value.split(/[?#]/)[0];
  const slash = value.indexOf("/");
  if (slash >= 0) {
    const prefix = value.slice(slash + 1);
    if (!/^\d{1,3}$/.test(prefix)) value = value.slice(0, slash);
  }
  // A port suffix never appears in a valid IP rule; drop it unless the value
  // already carries a CIDR prefix.
  if (!value.includes("/")) value = value.replace(/:\d+$/, "");
  return value || null;
}

/**
 * Process rules match an executable name and nothing else. A pasted full path
 * is reduced to its last segment, because the engine rejects anything holding a
 * separator or a drive letter.
 */
export function normalizeProcess(raw: string): string | null {
  let value = raw.trim();
  if (!value) return null;
  if (/[\\/]/.test(value)) value = value.split(/[\\/]/).pop() ?? "";
  value = value.trim();
  if (!value || /[:\\/*?"<>|]/.test(value)) return null;
  return value;
}

const IPV4 = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/;

function isIPv4(value: string): boolean {
  const match = IPV4.exec(value);
  return Boolean(match) && match!.slice(1).every((part) => Number(part) <= 255);
}

export type MatchKind = "process" | "domain" | "ip";

/**
 * Work out which kind of rule a value belongs to.
 *
 * The editor opens on the process tab, so pasting a URL there used to submit
 * `http://host/path` as a process name and earn a format error from the engine.
 * Deciding the kind from the value itself means the paste just works.
 */
export function detectMatchKind(raw: string): MatchKind | null {
  const value = raw.trim();
  if (!value) return null;

  // An executable, a full Windows path, or a drive-qualified path is a process
  // rule even though it contains separators or a colon.
  if (/\.exe$/i.test(value) || value.includes("\\") || /^[a-z]:[\\/]/i.test(value)) return "process";

  const looksLikeUrl = SCHEME.test(value) || /[/?#]/.test(value);
  if (looksLikeUrl) {
    const host = extractHost(value);
    if (!host) return null;
    // A URL pointing at a bare address belongs to the IP tab; the domain tab
    // would store a numeric host that reads like a mistake.
    return isIPv4(host) ? "ip" : "domain";
  }

  const withoutPrefix = value.replace(/\/\d{1,3}$/, "");
  if (isIPv4(withoutPrefix)) return "ip";

  // A bare hostname needs a dot and must survive normalisation to count as a
  // domain; everything else is treated as an executable name.
  if (value.includes(".") && !/\s/.test(value) && normalizeDomain(value)) return "domain";

  return "process";
}

/** Returns the value to submit for a given match type, or null when unusable. */
export function normalizeMatchValue(raw: string, type: "process" | "domain" | "ip"): string | null {
  if (type === "process") return normalizeProcess(raw);
  if (type === "domain") return normalizeDomain(raw);
  return normalizeIp(raw);
}
