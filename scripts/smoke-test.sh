#!/usr/bin/env sh
set -eu

API_URL="${API_URL:-http://localhost:8080}"
MAILPIT_URL="${MAILPIT_URL:-http://localhost:8025}"
EMAIL="smoke-$(date +%s)@example.com"

json_field() {
  FIELD="$1" node -e 'let data=""; process.stdin.on("data", chunk => data += chunk); process.stdin.on("end", () => console.log(JSON.parse(data)[process.env.FIELD]));'
}

auth_curl() {
  curl -fsS -H "Authorization: Bearer $ACCESS_TOKEN" "$@"
}

echo "Checking API readiness..."
curl -fsS "$API_URL/readyz" >/dev/null

echo "Requesting an email login code..."
curl -fsS -X POST "$API_URL/auth/request-code" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\"}" >/dev/null

CODE="$(curl -fsS "$MAILPIT_URL/api/v1/messages" | node -e 'let data=""; process.stdin.on("data", chunk => data += chunk); process.stdin.on("end", () => { const match = JSON.parse(data).messages[0]?.Snippet.match(/\b\d{6}\b/); if (!match) process.exit(1); console.log(match[0]); });')"

echo "Verifying the code and creating a session..."
SESSION_JSON="$(curl -fsS -X POST "$API_URL/auth/verify-code" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"code\":\"$CODE\"}")"
ACCESS_TOKEN="$(printf '%s' "$SESSION_JSON" | json_field accessToken)"
REFRESH_TOKEN="$(printf '%s' "$SESSION_JSON" | json_field refreshToken)"

echo "Saving and reloading the user profile..."
auth_curl -X POST "$API_URL/profile" \
  -H "Content-Type: application/json" \
  -d '{"displayName":"Smoke Tester","ageRange":"25-34"}' >/dev/null
auth_curl "$API_URL/me" | grep -q '"profileComplete":true'

echo "Checking topics and words..."
auth_curl "$API_URL/topics" | grep -q 'daily-basics'
auth_curl "$API_URL/topics/housing-renting/words" | grep -q 'woning'
auth_curl "$API_URL/knowledge-map" | grep -q '"availableWords":190'
RANGE_JSON="$(auth_curl "$API_URL/knowledge-map/ranges/301")"
printf '%s' "$RANGE_JSON" | grep -q 'woning'
RANGE_WORD_ID="$(printf '%s' "$RANGE_JSON" | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).words[0].id));')"
auth_curl -X POST "$API_URL/knowledge-map/words/$RANGE_WORD_ID/mark" \
  -H "Content-Type: application/json" \
  -d '{"action":"should_learn"}' \
  | grep -q '"status":"recorded"'

echo "Completing a practice answer..."
PRACTICE_JSON="$(auth_curl "$API_URL/sessions/next?topic=housing-renting")"
SESSION_ID="$(printf '%s' "$PRACTICE_JSON" | json_field id)"
QUESTION_ID="$(printf '%s' "$PRACTICE_JSON" | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).questions[0].id));')"
WORD_ID="$(printf '%s' "$PRACTICE_JSON" | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).questions[0].wordId));')"
auth_curl -X POST "$API_URL/sessions/$SESSION_ID/answer" \
  -H "Content-Type: application/json" \
  -d "{\"questionId\":\"$QUESTION_ID\",\"wordId\":\"$WORD_ID\",\"answer\":\"Already Knew\"}" \
  | grep -q '"isCorrect":true'
auth_curl "$API_URL/progress/summary" | grep -q '"dailyDone":1'

echo "Creating a mixed-category practice session..."
MIXED_JSON="$(auth_curl -X POST "$API_URL/sessions/next" \
  -H "Content-Type: application/json" \
  -d '{"topicSlugs":["housing-renting","healthcare"],"cefrLevels":[],"mode":"balanced"}')"
printf '%s' "$MIXED_JSON" | grep -q '"topicSlugs":\["housing-renting","healthcare"\]'
printf '%s' "$MIXED_JSON" | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => { if (JSON.parse(data).title !== "Housing & Renting + Healthcare") process.exit(1); });'

echo "Rotating and revoking the refresh token..."
REFRESH_JSON="$(curl -fsS -X POST "$API_URL/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refreshToken\":\"$REFRESH_TOKEN\"}")"
REFRESH_TOKEN="$(printf '%s' "$REFRESH_JSON" | json_field refreshToken)"
curl -fsS -X POST "$API_URL/auth/logout" \
  -H "Content-Type: application/json" \
  -d "{\"refreshToken\":\"$REFRESH_TOKEN\"}" >/dev/null
STATUS="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$API_URL/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refreshToken\":\"$REFRESH_TOKEN\"}")"
test "$STATUS" = "401"

echo "Smoke test passed."
