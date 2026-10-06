'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const { validate, matchesWorkspace, needsReload, confirmPending } = require('../state');
const now = Date.now();
const request = {version:1,id:'req-1',cwd:'/project',threadId:'thread-1',createdAt:new Date(now).toISOString(),daemonEpoch:'epoch-2',cliExecutable:'/trusted/launcher'};
test('request rejects stale, future, traversal and arbitrary executable', () => {
  assert.equal(validate(request,'req-1.json','/trusted/launcher',now),request);
  for (const patch of [{createdAt:new Date(now-120001).toISOString()},{createdAt:new Date(now+10001).toISOString()},{id:'../escape'},{threadId:'../escape'},{cliExecutable:'/tmp/evil'}]) assert.throws(()=>validate({...request,...patch},'req-1.json','/trusted/launcher',now));
});
test('workspace matching canonicalizes symlinks and isolates unrelated windows', async () => {
  const canonical = async p => ({'/alias':'/project'}[p] || p);
  assert.equal(await matchesWorkspace('/project',['/alias'],canonical),true);
  assert.equal(await matchesWorkspace('/project',['/other'],canonical),false);
});
test('active old daemon reloads once; persisted ticket survives reload failure', () => {
  assert.equal(needsReload(request,{epoch:'epoch-1'},true,false),true);
  assert.equal(needsReload(request,{reloadTicket:'req-1'},true,true),false);
  assert.equal(needsReload(request,{epoch:'epoch-2'},true,false),false);
  assert.equal(needsReload(request,{epoch:'epoch-2'},true,true),true);
  assert.equal(needsReload(request,undefined,false,true),false);
  assert.equal(needsReload(request,undefined,true,false),true);
});

test('launcher requires a configured absolute string and threads support 128 characters', () => {
  for (const launcher of [undefined, null, '', 42, 'relative']) assert.throws(() => validate(request, 'req-1.json', launcher, now));
  assert.doesNotThrow(() => validate({...request,threadId:'a'.repeat(128)}, 'req-1.json', '/trusted/launcher', now));
  assert.throws(() => validate({...request,threadId:'a'.repeat(129)}, 'req-1.json', '/trusted/launcher', now));
});
test('navigation recheck rejects removed, replaced or expired requests after activation', async () => {
  await assert.rejects(confirmPending('/request', request, '/trusted/launcher', async () => { throw Object.assign(Error('cancelled'), {code:'ENOENT'}); }, now));
  await assert.rejects(confirmPending('/request', request, '/trusted/launcher', async () => ({...request,threadId:'another-thread'}), now));
  await assert.rejects(confirmPending('/request', request, '/trusted/launcher', async () => request, now+120001));
  assert.equal(await confirmPending('/request', request, '/trusted/launcher', async () => request, now),request);
});
