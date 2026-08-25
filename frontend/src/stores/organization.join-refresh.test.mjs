import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./organization.ts', import.meta.url), 'utf8')

test('joining a space forces organization and shared-resource refreshes', () => {
  assert.match(
    source,
    /async function refreshAfterJoin\(\)[\s\S]*?await Promise\.all\(\[\s*fetchOrganizationsPromise,\s*fetchSharedKbPromise,\s*fetchSharedAgentsPromise,\s*\]\)[\s\S]*?fetchOrganizations\(\{ force: true \}\)[\s\S]*?fetchSharedKnowledgeBases\(\{ force: true \}\)[\s\S]*?fetchSharedAgents\(\{ force: true \}\)/,
  )

  const joinStart = source.indexOf('async function join(inviteCode: string)')
  const joinEnd = source.indexOf('/**', joinStart)
  const joinAction = joinStart >= 0 && joinEnd > joinStart
    ? source.slice(joinStart, joinEnd)
    : ''
  assert.match(joinAction, /await refreshAfterJoin\(\)/)
  assert.match(source, /\n    refreshAfterJoin,\n/)
})
