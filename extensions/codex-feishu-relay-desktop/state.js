'use strict';
const path = require('node:path');
function validate(request, filename, trustedLauncher, now = Date.now()) {
  if (!request || request.version !== 1 || !/^[a-zA-Z0-9_-]{1,100}$/.test(request.id || '') || filename !== `${request.id}.json`) throw Error('Invalid request identity');
  if (!path.isAbsolute(request.cwd || '') || !/^[a-zA-Z0-9_-]{1,128}$/.test(request.threadId || '') || typeof request.daemonEpoch !== 'string' || !request.daemonEpoch) throw Error('Invalid request target');
  if (typeof trustedLauncher !== 'string' || !path.isAbsolute(trustedLauncher) || request.cliExecutable !== trustedLauncher) throw Error('Untrusted Codex launcher');
  const age = now - Date.parse(request.createdAt);
  if (!Number.isFinite(age) || age < -10000 || age > 120000) throw Error('Request expired or invalid timestamp');
  return request;
}
async function matchesWorkspace(cwd, folders, realpath) {
  const target = await realpath(cwd);
  for (const folder of folders) { try { if (await realpath(folder) === target) return true; } catch {} }
  return false;
}
function needsReload(request, state, isActive, launcherChanged) {
  // A persisted ticket makes reload at most once, including crash/retry paths.
  if (state?.reloadTicket === request.id) return false;
  return isActive && (launcherChanged || state?.epoch !== request.daemonEpoch);
}
async function confirmPending(requestPath, expected, trustedLauncher, readRequest, now = Date.now()) {
  const current = await readRequest(requestPath);
  validate(current, `${expected.id}.json`, trustedLauncher, now);
  for (const key of ['id', 'cwd', 'threadId', 'daemonEpoch', 'createdAt', 'cliExecutable']) {
    if (current[key] !== expected[key]) throw Error('Request changed or cancelled');
  }
  return current;
}
module.exports = { validate, matchesWorkspace, needsReload, confirmPending };
