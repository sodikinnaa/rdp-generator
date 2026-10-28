import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createHmac } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const html = readFileSync(join(here, 'templates/index.html'), 'utf8');

assert.ok(!html.includes('{{'), 'Go html/template delimiters must not appear');
assert.ok(!html.includes('/download/exe'), 'local exe download must be removed');
assert.ok(html.includes('https://github.com/sodikinnaa/rdp-generator/releases/latest'), 'releases link required');

const inline = [...html.matchAll(/<script(?![^>]*src)[^>]*>([\s\S]*?)<\/script>/g)].map((m) => m[1]);
assert.equal(inline.length, 1, 'expected exactly one inline script');
new Function(inline[0]);

globalThis.isSecureContext = true;
globalThis.location = { hostname: 'localhost' };
const app = new Function(`${inline[0]}; return rdpSimpleApp();`)();
app.$nextTick = (fn) => fn();

assert.equal(app.secureContext, true);
assert.equal(app.publicInsecure, false);

// Storage tests: everything saved to localStorage
const storageMock = () => {
  const store = new Map();
  return {
    getItem: (k) => (store.has(k) ? store.get(k) : null),
    setItem: (k, v) => store.set(k, String(v)),
    removeItem: (k) => store.delete(k),
    clear: () => store.clear(),
    dump: () => Object.fromEntries(store.entries()),
  };
};

globalThis.sessionStorage = storageMock();
globalThis.localStorage = storageMock();

// Legacy migration check: keys in sessionStorage move to localStorage on init
globalThis.sessionStorage.setItem('repoOwner', 'legacy-owner');
globalThis.sessionStorage.setItem('repoName', 'legacy-repo');
globalThis.sessionStorage.setItem('githubToken', 'legacy-tok');
await app.init();
assert.equal(app.repoOwner, 'legacy-owner');
assert.equal(app.repoName, 'legacy-repo');
assert.equal(globalThis.localStorage.getItem('repoOwner'), 'legacy-owner');
assert.equal(globalThis.localStorage.getItem('githubToken'), 'legacy-tok');
assert.equal(globalThis.sessionStorage.getItem('repoOwner'), null);
assert.equal(globalThis.sessionStorage.getItem('githubToken'), null);
assert.equal(app.githubToken, 'legacy-tok');

// Save credentials saves everything to localStorage
app.githubToken = 'new-tok';
app.tailscaleKey = 'new-ts';
app.masterSecret = 'a'.repeat(32);
app.repoOwner = 'saved-owner';
app.repoName = 'saved-repo';
app.selectedOS = '2025';
app.user = { login: 'u' };
app.verifiedToken = 'new-tok';
app.repoValid = { valid: true };
app.validatedRepo = 'saved-owner/saved-repo';
app.sameTailnet = true;
app.magicDNS = true;
app.secretStored = true;
app.saveCredentials();

assert.equal(globalThis.localStorage.getItem('repoOwner'), 'saved-owner');
assert.equal(globalThis.localStorage.getItem('repoName'), 'saved-repo');
assert.equal(globalThis.localStorage.getItem('selectedOS'), '2025');
assert.equal(globalThis.localStorage.getItem('githubToken'), 'new-tok');
assert.equal(globalThis.localStorage.getItem('masterSecret'), 'a'.repeat(32));
assert.equal(globalThis.localStorage.getItem('tailscaleKey'), 'new-ts');

// Repository auto-list test
app.repositories = [
  { full_name: 'org/repo-one', private: false },
  { full_name: 'org/repo-two', private: true },
];
app.repoSearch = 'two';
assert.equal(app.filteredRepositories.length, 1);
assert.equal(app.filteredRepositories[0].full_name, 'org/repo-two');
app.repoSearch = '';
assert.equal(app.filteredRepositories.length, 2);

app.selectedRepo = 'org/repo-one';
await app.selectRepository();
assert.equal(app.repoOwner, 'org');
assert.equal(app.repoName, 'repo-one');

// Reset clears both
globalThis.confirm = () => true;
app.resetSession();
assert.equal(globalThis.localStorage.getItem('repoOwner'), null);
assert.equal(globalThis.localStorage.getItem('repoName'), null);
assert.equal(globalThis.localStorage.getItem('selectedOS'), null);
assert.equal(globalThis.localStorage.getItem('githubToken'), null);
assert.equal(globalThis.localStorage.getItem('masterSecret'), null);
assert.equal(app.repoOwner, '');
assert.equal(app.githubToken, '');

assert.equal(app.displayName({ name: 'rdp-v1-2022-office-pc' }), 'office-pc');
assert.equal(app.displayName({ name: 'rdp-v1-2021-keep' }), 'rdp-v1-2021-keep');
assert.equal(app.displayName({ display_name: 'rdp-v1-11-laptop', name: 'x' }), 'laptop');
assert.equal(app.statusLabel({ state: 'Available' }), 'Codespace aktif');
assert.equal(app.statusLabel({ state: 'Shutdown' }), 'Dihentikan');
assert.equal(app.statusLabel({ state: 'Weird' }), 'Status tidak diketahui');
assert.equal(app.safeURL('javascript:alert(1)'), '');
assert.equal(app.safeURL('https://u:p@example.com'), '');
assert.equal(app.safeURL('https://example.com/c'), 'https://example.com/c');

const secret = 'a'.repeat(32);
app.masterSecret = secret;
const derived = await app.derivePassword('rdp-v1-2022-x');
assert.equal(derived, `Aa1!${createHmac('sha256', secret).update('rdp-v1-2022-x').digest('hex').slice(0, 24)}`);
assert.equal(derived.length, 28);

assert.equal(app.canProvision, false, 'provisioning blocked before setup');
app.githubToken = 'tok';
app.user = { login: 'me' };
app.verifiedToken = 'tok';
app.repoOwner = 'o';
app.repoName = 'r';
app.repoValid = { valid: true, has_devcontainer: true };
app.validatedRepo = 'o/r';
app.tailscaleKey = 'tskey';
app.sameTailnet = true;
app.magicDNS = true;
app.secretStored = true;
assert.equal(app.allReady, true);
assert.equal(app.canProvision, false, 'canProvision requires saved session');
app.setupComplete = true;
assert.equal(app.canProvision, true);
app.secureContext = false;
assert.equal(app.canProvision, false, 'insecure context blocks provisioning');
app.secureContext = true;

app.instances = [
  { name: 'rdp-v1-2022-alpha', state: 'Available', repo_full_name: 'o/r' },
  { name: 'rdp-v1-2019-beta', state: 'Shutdown', repo_full_name: 'o/other' },
];
assert.deepEqual(app.filteredInstances.map((cs) => cs.name), ['rdp-v1-2022-alpha', 'rdp-v1-2019-beta']);
app.filter = 'active';
assert.deepEqual(app.filteredInstances.map((cs) => cs.name), ['rdp-v1-2022-alpha']);
app.filter = 'all';
app.search = 'BETA';
assert.deepEqual(app.filteredInstances.map((cs) => cs.name), ['rdp-v1-2019-beta']);
app.search = '';

assert.equal(app.canControl(app.instances[0], 'stop'), true);
assert.equal(app.canControl(app.instances[0], 'start'), false, 'cannot start a running machine');
assert.equal(app.canControl(app.instances[1], 'stop'), false, 'cannot stop a stopped machine');
assert.equal(app.canControl(app.instances[1], 'delete'), true);
app.pending['rdp-v1-2019-beta'] = 'start';
assert.equal(app.canControl(app.instances[1], 'start'), false, 'pending action blocks repeat');
app.pending = {};

const prior = app.instances.slice();
globalThis.fetch = async () => {
  throw new Error('network down');
};
assert.equal(await app.loadInstances(), false);
assert.equal(app.stale, true);
assert.deepEqual(app.instances, prior, 'failed refresh preserves prior instances');
globalThis.fetch = async () => ({ ok: true, json: async () => prior });
assert.equal(await app.loadInstances(), true);
assert.equal(app.stale, false);
assert.equal(app.loaded, true);
assert.equal(app.initialLoading, false);

app.rdpInstance = null;
await app.openRdp({ name: 'legacy-1', state: 'Available' });
assert.equal(app.rdpPassword, '');
assert.match(app.rdpReason, /Mesin lama/);
await app.openRdp({ name: 'rdp-v1-2022-alpha', state: 'Available', credential_scheme: 'hmac-sha256-v1' });
assert.equal(app.rdpPassword, await app.derivePassword('rdp-v1-2022-alpha'));
assert.equal(app.showPassword, false, 'password hidden by default');
app.masterSecret = 'short';
await app.openRdp({ name: 'rdp-v1-2022-alpha', state: 'Available', credential_scheme: 'hmac-sha256-v1' });
assert.equal(app.rdpPassword, '');
assert.match(app.rdpReason, /32 karakter/);
app.masterSecret = secret;

app.step = 1;
app.user = null;
app.verifiedToken = '';
assert.equal(app.stepReady, false, 'step 1 requires fresh verification');
app.user = { login: 'me' };
app.verifiedToken = 'tok';
assert.equal(app.stepReady, true);
app.step = 2;
app.repoValid = null;
assert.equal(app.stepReady, false, 'step 2 requires repository validation');

let copied = '';
Object.defineProperty(globalThis, 'navigator', { configurable: true, writable: true, value: { clipboard: { writeText: async (text) => { copied = text; } } } });
assert.equal(await app.copy('rahasia', 'Kunci pemulihan'), true);
assert.equal(copied, 'rahasia');
Object.defineProperty(globalThis, 'navigator', { configurable: true, writable: true, value: { clipboard: { writeText: async () => { throw new Error('denied'); } } } });
assert.equal(await app.copy('rahasia', 'Kunci pemulihan'), false);
assert.equal(app.alertError, true);

globalThis.btoa = (bin) => Buffer.from(bin, 'binary').toString('base64');
app.masterSecret = '';
app.generateSecret();
assert.ok(app.masterSecret.length >= 32, 'generated key must be at least 32 chars');
assert.match(app.masterSecret, /^[A-Za-z0-9_-]+$/, 'base64url only');
assert.equal(app.secretStored, false, 'new key resets stored confirmation');

const restored = new Function(`${inline[0]}; return rdpSimpleApp();`)();
for (const [key, value] of Object.entries({githubToken: 'saved-token', tailscaleKey: 'saved-ts', masterSecret: secret, repoOwner: 'owner', repoName: 'lab'})) localStorage.setItem(key, value);
restored.verifyUser = async function () { this.invalidateAuth(); this.user = {login: 'owner'}; this.verifiedToken = this.githubToken; };
restored.validateRepository = async function () { this.repoValid = {valid: true}; this.validatedRepo = 'owner/lab'; };
await restored.init();
assert.equal(restored.showWizard, false, 'saved credentials must not reopen wizard');
assert.equal(restored.canProvision, true, 'saved setup restored after validation');
restored.openSettings();
assert.equal(restored.showWizard, true, 'settings opens wizard manually');
restored.closeWizard();
assert.equal(restored.showWizard, false);
const failedRestore = new Function(`${inline[0]}; return rdpSimpleApp();`)();
globalThis.fetch = async () => { throw new Error('offline'); };
await failedRestore.init();
assert.equal(failedRestore.showWizard, false, 'verification failure must not reopen saved wizard');
assert.equal(failedRestore.canProvision, false);
localStorage.clear();
sessionStorage.clear();
const firstVisit = new Function(`${inline[0]}; return rdpSimpleApp();`)();
await firstVisit.init();
assert.equal(firstVisit.showWizard, true, 'new users receive onboarding');

console.log('index.html UI checks passed');
