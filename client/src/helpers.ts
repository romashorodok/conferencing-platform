const commonVersionIdentifier = /version\/(\d+(\.?_?\d+)+)/i;

export type DetectableBrowser = 'Chrome' | 'Firefox' | 'Safari';
export type DetectableOS = 'iOS' | 'macOS';

export type BrowserDetails = {
  name: DetectableBrowser;
  version: string;
  os?: DetectableOS;
};

let browserDetails: BrowserDetails | undefined;

export function getBrowser(userAgent?: string, force = true) {
  if (typeof userAgent === 'undefined' && typeof navigator === 'undefined') {
    return;
  }
  const ua = (userAgent ?? navigator.userAgent).toLowerCase();
  if (browserDetails === undefined || force) {
    const browser = browsersList.find(({ test }) => test.test(ua));
    browserDetails = browser?.describe(ua);
  }
  return browserDetails;
}

const browsersList = [
  {
    test: /firefox|iceweasel|fxios/i,
    describe(ua: string) {
      const browser: BrowserDetails = {
        name: 'Firefox',
        version: getMatch(/(?:firefox|iceweasel|fxios)[\s/](\d+(\.?_?\d+)+)/i, ua),
        os: ua.toLowerCase().includes('fxios') ? 'iOS' : undefined,
      };
      return browser;
    },
  },
  {
    test: /chrom|crios|crmo/i,
    describe(ua: string) {
      const browser: BrowserDetails = {
        name: 'Chrome',
        version: getMatch(/(?:chrome|chromium|crios|crmo)\/(\d+(\.?_?\d+)+)/i, ua),
        os: ua.toLowerCase().includes('crios') ? 'iOS' : undefined,
      };

      return browser;
    },
  },
  /* Safari */
  {
    test: /safari|applewebkit/i,
    describe(ua: string) {
      const browser: BrowserDetails = {
        name: 'Safari',
        version: getMatch(commonVersionIdentifier, ua),
        os: ua.includes('mobile/') ? 'iOS' : 'macOS',
      };

      return browser;
    },
  },
];

function getMatch(exp: RegExp, ua: string, id = 1) {
  const match = ua.match(exp);
  return (match && match.length >= id && match[id]) || '';
}

export function isFireFox(): boolean {
  return getBrowser()?.name === 'Firefox';
}

export function isChromiumBased(): boolean {
  return getBrowser()?.name === 'Chrome';
}

export function compareVersions(v1: string, v2: string): number {
  const parts1 = v1.split('.');
  const parts2 = v2.split('.');
  const k = Math.min(parts1.length, parts2.length);
  for (let i = 0; i < k; ++i) {
    const p1 = parseInt(parts1[i], 10);
    const p2 = parseInt(parts2[i], 10);
    if (p1 > p2) return 1;
    if (p1 < p2) return -1;
    if (i === k - 1 && p1 === p2) return 0;
  }
  if (v1 === '' && v2 !== '') {
    return -1;
  } else if (v2 === '') {
    return 1;
  }
  return parts1.length == parts2.length ? 0 : parts1.length < parts2.length ? -1 : 1;
}
