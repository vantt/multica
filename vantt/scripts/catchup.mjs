#!/usr/bin/env node
// Catchup: build the fork release branch `vantt/<tag>` from an upstream tag and
// apply the patch series in vantt/patches/series.toml. Pattern copied from
// vantt/herdr and vantt/obsidian-hybrid-search.
//
//   node vantt/scripts/catchup.mjs                 # tag in vantt/upstream.pin
//   node vantt/scripts/catchup.mjs --ref v0.7.2    # another upstream tag
//   node vantt/scripts/catchup.mjs --continue      # after fixing a conflicting patch
//   node vantt/scripts/catchup.mjs --abort
//   node vantt/scripts/catchup.mjs --dry-run       # validate series, print the plan
//
// Result: local branch vantt/<tag> = upstream tag + fork maintenance files + patches,
// and an annotated tag v<tag>-vantt.N (N goes up when the series changes on the same
// upstream tag). Nothing is pushed: CI (vantt-catchup.yml) pushes and builds images.
//
// Exit codes: 0 ok, 2 usage, 3 patch conflict, 5 patch test failed, 7 other error.
import { createHash } from 'node:crypto';
import { cpSync, existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync, spawnSync } from 'node:child_process';
import { parseArgs } from 'node:util';
import { parseSeries } from './lib/series.mjs';

const ROOT = execFileSync('git', ['rev-parse', '--show-toplevel'], { encoding: 'utf8' }).trim();
const UPSTREAM_URL = 'https://github.com/multica-ai/multica.git';
const STATE = path.join(ROOT, '.git', 'vantt-catchup-state.json');
const SERIES = 'vantt/patches/series.toml';
const EXIT = { ok: 0, usage: 2, conflict: 3, tests: 5, error: 7 };
// Fork-owned paths copied onto the release branch (the same set vantt-pr-check allows on main).
const FORK_PATHS = ['vantt', '.claude/skills', '.github/workflows'];
const FORK_FILE = /^(vantt\/|\.claude\/skills\/multica-|\.github\/workflows\/vantt-)/;

const git = (args, opts = {}) => execFileSync('git', args, { cwd: ROOT, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], ...opts }).trim();
const tryGit = (args) => spawnSync('git', args, { cwd: ROOT, encoding: 'utf8' });
const log = (m) => console.log(`[catchup] ${m}`);
const die = (code, m) => { console.error(`[catchup] ${m}`); process.exit(code); };

const { values: o } = parseArgs({
  options: {
    ref: { type: 'string' }, continue: { type: 'boolean' }, abort: { type: 'boolean' },
    'dry-run': { type: 'boolean' }, 'skip-tests': { type: 'boolean' }, help: { type: 'boolean', short: 'h' },
  },
});
if (o.help) { console.log(readFileSync(new URL(import.meta.url)).toString().split('\n').slice(1, 17).join('\n')); process.exit(0); }

function ensureUpstream() {
  if (tryGit(['remote', 'get-url', 'upstream']).status !== 0) {
    git(['remote', 'add', 'upstream', UPSTREAM_URL]);
    git(['remote', 'set-url', '--push', 'upstream', 'DISABLE']);
  }
  git(['fetch', '--quiet', '--tags', 'upstream', '+refs/heads/main:refs/remotes/upstream/main']);
}

// Hash of what the release is made of: the upstream tag, the series and every patch file.
function inputsHash(tag, patches) {
  const h = createHash('sha256').update(tag).update(readFileSync(path.join(ROOT, SERIES)));
  for (const p of patches) h.update(readFileSync(path.join(ROOT, 'vantt/patches', p.patch_file)));
  return h.digest('hex').slice(0, 16);
}

function nextVersion(tag, hash) {
  const tags = git(['tag', '-l', `${tag}-vantt.*`]).split('\n').filter(Boolean);
  let max = 0;
  for (const t of tags) {
    const n = Number(t.split('-vantt.')[1]);
    const msg = git(['tag', '-l', '--format=%(contents)', t]);
    if (msg.includes(`inputs: ${hash}`)) return { version: t, existing: true };
    if (n > max) max = n;
  }
  return { version: `${tag}-vantt.${max + 1}`, existing: false };
}

function loadSeries() {
  const file = path.join(ROOT, SERIES);
  if (!existsSync(file)) die(EXIT.usage, `${SERIES} missing`);
  const all = parseSeries(readFileSync(file, 'utf8'));
  for (const p of all) {
    for (const k of ['id', 'name', 'patch_file', 'metadata_file']) if (!p[k]) die(EXIT.usage, `series: patch ${p.id || '?'} needs ${k}`);
    if (!existsSync(path.join(ROOT, 'vantt/patches', p.patch_file))) die(EXIT.usage, `series: ${p.patch_file} missing`);
    if (!existsSync(path.join(ROOT, 'vantt/patches', p.metadata_file))) die(EXIT.usage, `series: ${p.metadata_file} missing`);
  }
  return all.filter((p) => p.upstream_status !== 'merged');
}

function saveState(s) { writeFileSync(STATE, JSON.stringify(s, null, 2)); }
function loadState() { return existsSync(STATE) ? JSON.parse(readFileSync(STATE, 'utf8')) : null; }

function runTest(p) {
  if (o['skip-tests'] || !p.test_command) return;
  log(`test ${p.id}: ${p.test_command}`);
  const r = spawnSync('bash', ['-lc', p.test_command], { cwd: ROOT, stdio: 'inherit' });
  if (r.status !== 0) die(EXIT.tests, `test of patch ${p.id} failed; branch left as is`);
}

function applyFrom(state, patches) {
  for (let i = state.next; i < patches.length; i++) {
    const p = patches[i];
    log(`apply ${p.id}: ${p.name}`);
    const r = tryGit(['am', '--3way', '--keep-cr', path.join(ROOT, 'vantt/patches', p.patch_file)]);
    if (r.status !== 0) {
      saveState({ ...state, next: i, conflict: p.id });
      const files = git(['diff', '--name-only', '--diff-filter=U']);
      die(EXIT.conflict, `CONFLICT in patch ${p.id}\nfiles:\n${files || '(see git am output)'}\n` +
        `spec: vantt/patches/${p.metadata_file} (invariants and how to rewrite)\n` +
        `fix, git add, then: node vantt/scripts/catchup.mjs --continue   (or --abort)`);
    }
    runTest(p);
    state.next = i + 1; saveState(state);
  }
}

function finish(state, patches) {
  const { version, existing } = nextVersion(state.tag, state.hash);
  if (existing) log(`inputs unchanged; release ${version} already exists`);
  else {
    git(['tag', '-a', version, '-m', `fork release ${version}\n\nupstream: ${state.tag}\ninputs: ${state.hash}\npatches: ${patches.map((p) => p.id).join(', ') || '(none)'}`]);
    log(`tagged ${version}`);
  }
  rmSync(STATE, { force: true });
  git(['checkout', '--quiet', state.from]);
  console.log(JSON.stringify({ status: 'SUCCESS', branch: state.branch, version, existing, patches: patches.map((p) => p.id) }));
}

function main() {
  if (o.abort) {
    const s = loadState(); if (!s) die(EXIT.usage, 'nothing to abort');
    tryGit(['am', '--abort']); git(['checkout', '--quiet', s.from]); tryGit(['branch', '-D', s.branch]);
    rmSync(STATE, { force: true }); log('aborted'); return;
  }
  if (o.continue) {
    const s = loadState(); if (!s) die(EXIT.usage, 'nothing to continue');
    const patches = loadSeries();
    const r = tryGit(['am', '--continue']);
    if (r.status !== 0) die(EXIT.conflict, `git am --continue failed:\n${r.stderr}`);
    runTest(patches[s.next]); s.next += 1; s.conflict = null; saveState(s);
    applyFrom(s, patches); finish(s, patches); return;
  }

  if (tryGit(['diff', '--quiet']).status !== 0 || tryGit(['diff', '--cached', '--quiet']).status !== 0) die(EXIT.usage, 'working tree is dirty');
  if (loadState()) die(EXIT.usage, 'a catchup is in progress: --continue or --abort');
  const patches = loadSeries();
  const tag = o.ref || readFileSync(path.join(ROOT, 'vantt/upstream.pin'), 'utf8').trim();
  if (!/^v\d+\.\d+\.\d+$/.test(tag)) die(EXIT.usage, `not a stable upstream tag: ${tag}`);
  const hash = inputsHash(tag, patches);
  if (o['dry-run']) {
    console.log(JSON.stringify({ tag, hash, branch: `vantt/${tag}`, patches: patches.map((p) => ({ id: p.id, status: p.upstream_status || 'unsubmitted' })) }, null, 2));
    return;
  }
  ensureUpstream();
  if (tryGit(['rev-parse', '--verify', `refs/tags/${tag}`]).status !== 0) die(EXIT.usage, `upstream tag ${tag} not found`);
  if (tryGit(['merge-base', '--is-ancestor', tag, 'upstream/main']).status !== 0) die(EXIT.usage, `${tag} is not on upstream main`);

  const from = git(['rev-parse', '--abbrev-ref', 'HEAD']);
  const branch = `vantt/${tag}`;
  // Fork files come from the current checkout (main), copied before switching.
  const tmp = mkdtempSync(path.join(os.tmpdir(), 'vantt-fork-'));
  for (const rel of FORK_PATHS) if (existsSync(path.join(ROOT, rel))) cpSync(path.join(ROOT, rel), path.join(tmp, rel), { recursive: true });

  log(`branch ${branch} from ${tag}`);
  git(['checkout', '--quiet', '-B', branch, tag]);
  // Upstream workflows never run on the fork's release branch; only vantt-* ones.
  for (const f of git(['ls-files', '.github/workflows']).split('\n').filter(Boolean)) git(['rm', '--quiet', f]);
  for (const rel of FORK_PATHS) if (existsSync(path.join(tmp, rel))) cpSync(path.join(tmp, rel), path.join(ROOT, rel), { recursive: true });
  for (const f of git(['ls-files', '--others', '--exclude-standard']).split('\n').filter(Boolean)) if (FORK_FILE.test(f)) git(['add', f]);
  git(['commit', '--quiet', '--allow-empty', '-m', 'chore(vantt): fork maintenance files; drop upstream workflows']);
  rmSync(tmp, { recursive: true, force: true });

  const state = { tag, branch, from, hash, next: 0, conflict: null };
  saveState(state);
  applyFrom(state, patches);
  finish(state, patches);
}

try { main(); } catch (e) { die(EXIT.error, e.stderr?.toString() || e.message); }
