import assert from 'node:assert/strict'
import { test } from 'node:test'
import { copyText } from './clipboard.ts'

test('copies with the Clipboard API when it is available', async () => {
  const writes: string[] = []
  const previousNavigator = globalThis.navigator

  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: { clipboard: { writeText: async (text: string) => writes.push(text) } },
  })

  try {
    await copyText('mcp config')
    assert.deepEqual(writes, ['mcp config'])
  } finally {
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: previousNavigator,
    })
  }
})

test('falls back to a textarea when Clipboard API is unavailable', async () => {
  const copied: string[] = []
  const previousNavigator = globalThis.navigator
  const previousDocument = globalThis.document
  const textarea = {
    value: '',
    style: {} as Record<string, string>,
    setAttribute: () => undefined,
    select: () => undefined,
  }
  const body = {
    appendChild: (node: typeof textarea) => { textarea.value = node.value },
    removeChild: () => undefined,
  }

  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: {} })
  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: {
      body,
      createElement: () => textarea,
      execCommand: (command: string) => {
        if (command === 'copy') copied.push(textarea.value)
        return true
      },
    },
  })

  try {
    await copyText('fallback config')
    assert.deepEqual(copied, ['fallback config'])
  } finally {
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: previousNavigator,
    })
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: previousDocument,
    })
  }
})

test('falls back when Clipboard API rejects in an insecure context', async () => {
  const copied: string[] = []
  const previousNavigator = globalThis.navigator
  const previousDocument = globalThis.document
  const textarea = {
    value: '',
    style: {} as Record<string, string>,
    setAttribute: () => undefined,
    select: () => undefined,
  }

  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: { clipboard: { writeText: async () => { throw new Error('NotAllowedError') } } },
  })
  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: {
      body: {
        appendChild: () => undefined,
        removeChild: () => undefined,
      },
      createElement: () => textarea,
      execCommand: (command: string) => {
        if (command === 'copy') copied.push(textarea.value)
        return true
      },
    },
  })

  try {
    await copyText('insecure context config')
    assert.deepEqual(copied, ['insecure context config'])
  } finally {
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: previousNavigator,
    })
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: previousDocument,
    })
  }
})
