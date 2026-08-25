import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./OrganizationList.vue', import.meta.url), 'utf8')

test('both direct join entry points await the post-join refresh', () => {
  const inviteStart = source.indexOf('async function confirmJoinOrganization()')
  const inviteEnd = source.indexOf('async function doPreviewFromInput()', inviteStart)
  const inviteJoin = source.slice(inviteStart, inviteEnd)
  assert.match(inviteJoin, /await orgStore\.join\(inviteCode\.value\)/)
  assert.doesNotMatch(inviteJoin, /joinOrganization\(\{ invite_code/)

  const searchStart = source.indexOf('async function joinBySearchOrg()')
  const searchEnd = source.indexOf('// Lifecycle', searchStart)
  const searchJoin = source.slice(searchStart, searchEnd)
  assert.match(searchJoin, /await orgStore\.refreshAfterJoin\(\)/)
})
