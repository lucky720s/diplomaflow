BEGIN;

UPDATE action_registry
SET config_schema = '{
  "type":"object",
  "required":["title","message"],
  "properties":{
    "title":{"type":"string"},
    "message":{"type":"string"},
    "link":{"type":"string"},
    "type":{"type":"string","default":"WORKFLOW"},
    "recipients":{"type":"string","enum":["student","team","supervisor","explicit"],"default":"student"},
    "explicit_user_ids":{"type":"array","items":{"type":"integer"}}
  }
}'::jsonb
WHERE id = 'SEND_NOTIFICATION';

COMMIT;
