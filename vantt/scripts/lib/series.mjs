// Minimal parser for vantt/patches/series.toml: [[patches]] tables whose values are
// strings ("..." or '...'), booleans, or arrays of strings (may span lines). Comments ok.
export function parseSeries(text) {
  const out = [];
  let cur = null, pendingKey = null, buf = '';
  const value = (raw) => {
    raw = raw.trim();
    if (raw === 'true' || raw === 'false') return raw === 'true';
    if (raw.startsWith('[')) return [...raw.matchAll(/"((?:[^"\\]|\\.)*)"|'([^']*)'/g)].map((m) => m[1] ?? m[2]);
    const m = raw.match(/^"((?:[^"\\]|\\.)*)"|^'([^']*)'/);
    if (!m) throw new Error(`series.toml: cannot read value ${raw}`);
    return (m[1] ?? m[2]).replace(/\\"/g, '"');
  };
  for (const rawLine of text.split('\n')) {
    const line = rawLine.replace(/^\s*#.*$/, '').replace(/\s+#[^"']*$/, '');
    if (pendingKey) {
      buf += ' ' + line;
      if (line.includes(']')) { cur[pendingKey] = value(buf); pendingKey = null; buf = ''; }
      continue;
    }
    if (!line.trim()) continue;
    if (line.trim() === '[[patches]]') { cur = {}; out.push(cur); continue; }
    const m = line.match(/^\s*([a-z_]+)\s*=\s*(.*)$/);
    if (!m || !cur) throw new Error(`series.toml: unexpected line: ${rawLine}`);
    if (m[2].trim().startsWith('[') && !m[2].includes(']')) { pendingKey = m[1]; buf = m[2]; continue; }
    cur[m[1]] = value(m[2]);
  }
  const ids = new Set();
  for (const p of out) {
    if (ids.has(p.id)) throw new Error(`series.toml: duplicate id ${p.id}`);
    ids.add(p.id);
  }
  return out;
}
