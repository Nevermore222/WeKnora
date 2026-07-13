# Skills API

[Back to API index](./README.md)

Skills are scanned from the configured preloaded skills directory. The API is read-only in this phase so the frontend can build a Skill Studio and `/skill` picker without exposing install or mutation operations yet.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/skills` | List discovered skills with lightweight script summaries |
| GET | `/skills/{name}` | Get one skill's instructions and resource summary |
| GET | `/skills/{name}/files/{path}` | Read a file inside one skill directory |

## GET `/skills`

Returns all discovered skills. `skills_available` is `true` only when the sandbox mode is enabled.

```bash
curl --location 'http://localhost:8080/api/v1/skills' \
  --header 'X-API-Key: sk-xxxxx'
```

```json
{
  "success": true,
  "skills_available": true,
  "data": [
    {
      "name": "officecli-document-editing",
      "description": "Create or modify Office files through the OfficeCLI bridge.",
      "source": "preloaded",
      "status": "enabled",
      "scripts": [
        {
          "path": "scripts/officecli_bridge.py",
          "language": "python"
        }
      ]
    }
  ]
}
```

## GET `/skills/{name}`

Loads a skill's full `SKILL.md` instructions plus a summary of files and executable scripts.

```bash
curl --location 'http://localhost:8080/api/v1/skills/officecli-document-editing' \
  --header 'X-API-Key: sk-xxxxx'
```

```json
{
  "success": true,
  "data": {
    "name": "officecli-document-editing",
    "description": "Create or modify Office files through the OfficeCLI bridge.",
    "source": "preloaded",
    "status": "enabled",
    "instructions": "Use this skill for Office document editing...",
    "scripts": [
      {
        "path": "scripts/officecli_bridge.py",
        "language": "python"
      }
    ],
    "files": [
      {
        "path": "SKILL.md",
        "is_script": false
      },
      {
        "path": "scripts/officecli_bridge.py",
        "is_script": true
      }
    ]
  }
}
```

## GET `/skills/{name}/files/{path}`

Reads one file from inside the skill directory. The path must be relative to that skill; absolute paths and directory traversal are rejected.

```bash
curl --location 'http://localhost:8080/api/v1/skills/officecli-document-editing/files/scripts/officecli_bridge.py' \
  --header 'X-API-Key: sk-xxxxx'
```

```json
{
  "success": true,
  "data": {
    "path": "scripts/officecli_bridge.py",
    "content": "print('example')\n",
    "is_script": true
  }
}
```
