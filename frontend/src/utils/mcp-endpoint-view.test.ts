import assert from 'node:assert/strict'
import { test } from 'node:test'
import { buildMcpEndpointUrl, resolveMcpPublicOrigin } from './mcp-endpoint-view.ts'

test('builds a public MCP URL from the frontend origin and endpoint path', () => {
  assert.equal(
    buildMcpEndpointUrl('https://xelora.example.com/', '/mcp/endpoint-123'),
    'https://xelora.example.com/mcp/endpoint-123',
  )
})

test('normalizes a path when the endpoint API returns a relative endpoint id', () => {
  assert.equal(
    buildMcpEndpointUrl('http://localhost:8081', 'mcp/endpoint-123'),
    'http://localhost:8081/mcp/endpoint-123',
  )
})

test('uses the configured public IP instead of the browser localhost origin', () => {
  assert.equal(
    resolveMcpPublicOrigin('http://localhost:8081', 'http://192.168.9.177'),
    'http://192.168.9.177',
  )
})

test('keeps the browser origin when no public origin is configured', () => {
  assert.equal(
    resolveMcpPublicOrigin('https://knavis.example.com', ''),
    'https://knavis.example.com',
  )
})
