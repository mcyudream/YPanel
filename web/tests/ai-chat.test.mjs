// Run with node --experimental-vm-modules --test tests/ai-chat.test.mjs.
import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
// Node's built-in runner keeps these regressions dependency-free.
// eslint-disable-next-line test/no-import-node-test
import test from 'node:test'
import vm from 'node:vm'
import { createPinia, defineStore, setActivePinia } from 'pinia'
import ts from 'typescript'
import * as vue from 'vue'

const sleep = ms => new Promise(resolve => setTimeout(resolve, ms))
const defaults = () => ({ providerId: 1, model: 'mock', mode: 'confirm', maxRounds: 100, contextLimit: 32768, maxOutputTokens: 4096, autoCompact: true, compactThreshold: 0.8, retainTurns: 4, summaryProviderId: 0, summaryModel: '', systemPrompt: '', skills: [], memory: true, knowledge: true, tools: true })
const snapshot = (id = 1, content = 'base', status = 'running') => ({ conversationId: id, runId: `run-${id}`, title: `chat-${id}`, version: 1, status, lastEventId: 0, total: 2, hasMore: false, messages: [{ id: `u-${id}`, role: 'user', content: 'hi', steps: [] }, { id: `a-${id}`, role: 'assistant', content, pending: status === 'running', steps: [], segments: [{ type: 'text', text: content }] }], settings: defaults(), context: { tokens: 100, limit: 32768, outputReserve: 4096, estimated: true } })

async function fixture() {
  setActivePinia(createPinia())
  const account = vue.reactive({ token: 'test-user' })
  const sockets = []
  const snapshots = new Map([[1, snapshot(1)], [2, snapshot(2, 'other', 'done')]])
  let requests = 0
  const api = {
    defaults: async () => defaults(),
    snapshot: async (id) => {
      requests++
      return structuredClone(snapshots.get(id))
    },
    start: async () => snapshot(3, '', 'running'),
    stop: async () => {},
  }
  const read = (id, runId, offset, signal, receive) => new Promise((resolve, reject) => {
    sockets.push({ id, runId, offset, signal, receive, resolve, reject })
    signal.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')), { once: true })
  })
  const context = vm.createContext({ console, setTimeout, clearTimeout, AbortController, crypto, URL, DOMException, structuredClone })
  const modules = {
    vue,
    'pinia': { defineStore },
    '@/api/modules/ai': { default: { providers: async () => [] } },
    '@/api/modules/aiChat': { aiChatApi: api, defaultChatSettings: defaults, readChatStream: read },
    '@/store/modules/app/account': { useAppAccountStore: () => account },
  }
  const source = await fs.readFile(new URL('../apps/core-arco-design-vue/src/store/modules/aiChat.ts', import.meta.url), 'utf8')
  const code = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext } }).outputText
  const module = new vm.SourceTextModule(code, { context })
  await module.link(async (name) => {
    const values = modules[name]
    assert.ok(values, name)
    return new vm.SyntheticModule(Object.keys(values), function () {
      for (const [k, v] of Object.entries(values)) {
        this.setExport(k, v)
      }
    }, { context })
  })
  await module.evaluate()
  return { store: module.namespace.useAiChatStore(), api, sockets, snapshots, account, requests: () => requests }
}

test('two views share one stream; completed reopen does not duplicate the answer', async () => {
  const f = await fixture()
  const e = f.store.entry(1)
  const leaveA = f.store.observe(e)
  const leaveB = f.store.observe(e)
  await sleep(10)
  assert.equal(f.sockets.length, 1)
  const socket = f.sockets[0]
  socket.receive({ runId: 'run-1', eventId: 1, content: ' delta' })
  socket.receive({ runId: 'run-1', eventId: 1, content: ' duplicate' })
  await sleep(50)
  assert.equal(e.messages.at(-1).content, 'base delta')
  leaveA()
  assert.equal(socket.signal.aborted, false)
  socket.receive({ runId: 'run-1', eventId: 2, done: true, status: 'done' })
  socket.resolve()
  await sleep(50)
  f.snapshots.set(1, snapshot(1, 'base delta', 'done'))
  leaveB()
  const leaveC = f.store.observe(e)
  await sleep(10)
  assert.equal(e.messages.length, 2)
  assert.equal(e.messages.at(-1).content, 'base delta')
  assert.equal(f.sockets.length, 1)
  leaveC()
})

test('switching aborts the viewer without stopping the task or mixing drafts', async () => {
  const f = await fixture()
  let stops = 0
  f.api.stop = async () => {
    stops++
  }
  const a = f.store.entry(1)
  const b = f.store.entry(2)
  a.draft = 'draft A'
  b.draft = 'draft B'
  const leave = f.store.observe(a)
  await sleep(10)
  leave()
  const enter = f.store.observe(b)
  await sleep(10)
  f.sockets[0].receive({ runId: 'run-1', eventId: 1, content: 'stale callback' })
  await sleep(50)
  assert.equal(a.messages.at(-1).content, 'base')
  assert.equal(b.messages.at(-1).content, 'other')
  assert.equal(a.draft, 'draft A')
  assert.equal(b.draft, 'draft B')
  assert.equal(stops, 0)
  enter()
})

test('cursor gaps abort promptly and fetch an atomic snapshot', async () => {
  const f = await fixture()
  const e = f.store.entry(1)
  const leave = f.store.observe(e)
  await sleep(10)
  const s = snapshot(1, 'atomic recovery')
  s.lastEventId = 8
  f.snapshots.set(1, s)
  f.sockets[0].receive({ runId: 'run-1', eventId: 8, content: 'gap' })
  await sleep(100)
  assert.equal(f.sockets[0].signal.aborted, true)
  assert.ok(f.requests() >= 2)
  assert.equal(e.messages.at(-1).content, 'atomic recovery')
  assert.equal(e.cursor, 8)
  leave()
})

test('failed send retains its draft and idempotency key', async () => {
  const f = await fixture()
  const e = f.store.entry(f.store.create())
  e.draft = 'please keep this'
  const keys = []
  f.api.start = async (body) => {
    keys.push(body.requestId)
    if (keys.length === 1) {
      throw new Error('network')
    }
    return snapshot(3, '', 'done')
  }
  await assert.rejects(f.store.send(e), /network/)
  assert.equal(e.draft, 'please keep this')
  await f.store.send(e)
  assert.equal(keys[0], keys[1])
  assert.equal(e.draft, '')
  assert.equal(e.id, 3)
})

test('logout clears conversations and rejects in-flight data from the old account', async () => {
  const f = await fixture()
  const e = f.store.entry(f.store.create())
  e.draft = 'private'
  let complete
  f.api.start = () => new Promise(resolve => complete = resolve)
  const pending = f.store.send(e)
  f.account.token = ''
  await vue.nextTick()
  complete(snapshot(3, 'previous user data', 'done'))
  await assert.rejects(pending, /Account changed/)
  assert.deepEqual(Object.keys(f.store.entries), [])
})

test('a stale HTTP snapshot cannot overwrite a newer run or rewind its cursor', async () => {
  const f = await fixture()
  const e = f.store.entry(1)
  const latest = snapshot(1, 'latest answer')
  latest.version = 2
  latest.lastEventId = 20
  f.store.applySnapshot(e, latest)
  f.store.applySnapshot(e, snapshot(1, 'old history'))
  const oldCursor = snapshot(1, 'old cursor')
  oldCursor.version = 2
  oldCursor.lastEventId = 8
  f.store.applySnapshot(e, oldCursor)
  assert.equal(e.messages.at(-1).content, 'latest answer')
  assert.equal(e.cursor, 20)
})
