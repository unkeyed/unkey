
import { extractLongTokenHash } from "prefixed-api-key"



const key = "re_xxx_xxxx"

console.log(extractLongTokenHash(key))





/*
curl -XPOST 'http://localhost:7070/v2/keys.verifyKey' \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer xxxx" \
  -d '{
    "key": "xxx",
    "migrationId": "resend"
  }'

*/
