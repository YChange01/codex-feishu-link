'use strict';
const vscode = require('vscode');
const fs = require('node:fs/promises');
const constants = require('node:fs').constants;
const path = require('node:path');
const os = require('node:os');
const { validate, matchesWorkspace, needsReload, confirmPending } = require('./state');
const directory = path.join(os.homedir(), '.codex', 'feishu-desktop', 'requests');
const stateKey = 'codexRemoteDesktop.recovery';
function privateOwned(stat) {
  return (typeof process.getuid !== 'function' || stat.uid === process.getuid()) && !(stat.mode & 0o077);
}
async function readPrivateRequest(filename) {
  const handle = await fs.open(filename, constants.O_RDONLY | constants.O_NOFOLLOW);
  try {
    const stat = await handle.stat();
    if (!stat.isFile() || !privateOwned(stat) || stat.size > 16384) throw Error('Invalid request permissions');
    return JSON.parse(await handle.readFile('utf8'));
  } finally { await handle.close(); }
}
async function atomicResult(request, result) {
  const destination = path.join(directory, `${request.id}.result.json`);
  const temporary = `${destination}.${process.pid}.tmp`;
  await fs.writeFile(temporary, JSON.stringify({ id: request.id, threadId: request.threadId, cwd: request.cwd, ...result }), { mode: 0o600, flag: 'wx' });
  await fs.rename(temporary, destination);
}
function activate(context) {
  let busy = false, disposed = false, reloading = false;
  const output = vscode.window.createOutputChannel('Codex Feishu Relay Desktop');
  context.subscriptions.push(output);
  async function processRequest(filename, trustedLauncher) {
    const requestPath = path.join(directory, filename);
    let handle, lock, request;
    const lockPath = path.join(directory, `${filename.slice(0, -5)}.claim`);
    try {
      handle = await fs.open(requestPath, constants.O_RDONLY | constants.O_NOFOLLOW);
      const stat = await handle.stat();
      if (!stat.isFile() || !privateOwned(stat) || stat.size > 16384) return;
      request = JSON.parse(await handle.readFile('utf8'));
      const folders = (vscode.workspace.workspaceFolders || []).filter(f => f.uri.scheme === 'file').map(f => f.uri.fsPath);
      if (typeof request.cwd !== 'string' || !await matchesWorkspace(request.cwd, folders, fs.realpath)) return;
      if (!/^[a-zA-Z0-9_-]{1,100}$/.test(request.id || '')) throw Error('Invalid request identity');
      try { await fs.access(path.join(directory, `${request.id}.result.json`)); return; } catch {}
      validate(request, filename, trustedLauncher);
      // Independent windows for the same project must not reload/open concurrently.
      try {
        lock = await fs.open(lockPath, 'wx', 0o600);
        await lock.writeFile(JSON.stringify({ id: request.id, createdAt: new Date().toISOString(), pid: process.pid }));
      } catch (error) {
        if (error.code !== 'EEXIST') throw error;
        const old = await fs.lstat(lockPath);
        if (!old.isFile() || !privateOwned(old)) return;
        const pending = context.workspaceState.get(stateKey);
        if (pending?.reloadTicket === request.id) {
          lock = await fs.open(lockPath, constants.O_WRONLY | constants.O_NOFOLLOW);
        } else {
          if (Date.now() - old.mtimeMs > 120000) await fs.unlink(lockPath);
          return;
        }
      }
      const extension = vscode.extensions.getExtension('openai.chatgpt');
      if (!extension) throw Error('Codex extension openai.chatgpt is not installed');
      const launcherStat = await fs.stat(trustedLauncher);
      if (!launcherStat.isFile() || !(launcherStat.mode & 0o111)) throw Error('Trusted shared Codex launcher is not executable');
      const config = vscode.workspace.getConfiguration('chatgpt');
      const changed = config.get('cliExecutable') !== trustedLauncher;
      const recovery = context.workspaceState.get(stateKey);
      const reload = needsReload(request, recovery, extension.isActive, changed);
      if (changed) await config.update('cliExecutable', trustedLauncher, vscode.ConfigurationTarget.Global);
      if (reload) {
        await confirmPending(requestPath, request, trustedLauncher, readPrivateRequest);
        await context.workspaceState.update(stateKey, { ...recovery, reloadTicket: request.id, pending: request });
        await lock.close(); lock = null;
        await confirmPending(requestPath, request, trustedLauncher, readPrivateRequest);
        reloading = true;
        await vscode.commands.executeCommand('workbench.action.reloadWindow');
        return;
      }
      await extension.activate();
      const uri = vscode.Uri.from({ scheme: 'openai-codex', authority: 'route', path: `/local/${request.threadId}` });
      await confirmPending(requestPath, request, trustedLauncher, readPrivateRequest);
      await vscode.commands.executeCommand('vscode.openWith', uri, 'chatgpt.conversationEditor');
      await context.workspaceState.update(stateKey, { epoch: request.daemonEpoch, threadId: request.threadId });
      await atomicResult(request, { status: 'opened' });
      output.appendLine(`Opened ${request.threadId}`);
    } catch (error) {
      reloading = false;
      output.appendLine(`Desktop request failed: ${error.message}`);
      // A new broker request ID permits a bounded, explicit retry after failure.
      if (request && /^[a-zA-Z0-9_-]{1,100}$/.test(request.id || '') && filename === `${request.id}.json`) {
        try { await atomicResult(request, { status: 'failed', error: String(error.message) }); } catch {}
      }
    } finally {
      if (handle) await handle.close();
      if (lock) { await lock.close(); await fs.unlink(lockPath).catch(() => {}); }
    }
  }
  async function poll() {
    if (busy || disposed || reloading) return;
    busy = true;
    try {
      const dir = await fs.lstat(directory);
      if (!dir.isDirectory() || !privateOwned(dir)) return;
      const trustedLauncher = vscode.workspace.getConfiguration('codexRemoteDesktop').get('cliExecutable');
      for (const filename of (await fs.readdir(directory)).sort()) {
        if (/^[a-zA-Z0-9_-]{1,100}\.json$/.test(filename)) await processRequest(filename, trustedLauncher);
        if (reloading || disposed) break;
      }
    } catch (error) { if (error.code !== 'ENOENT') output.appendLine(error.message); }
    finally { busy = false; }
  }
  const timer = setInterval(poll, 1000);
  context.subscriptions.push({ dispose() { disposed = true; clearInterval(timer); } });
  void poll();
}
module.exports = { activate };
