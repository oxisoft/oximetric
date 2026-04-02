#!/usr/bin/env bash
set -euo pipefail

BASE="${OXIMETRIC_URL:-http://localhost:6940}"
TOKEN="${OXIMETRIC_TOKEN:-}"

if [ -z "$TOKEN" ]; then
  echo "Usage: OXIMETRIC_TOKEN=<token> ./scripts/seed.sh"
  echo "  or:  OXIMETRIC_TOKEN=<token> OXIMETRIC_URL=http://host:port ./scripts/seed.sh"
  exit 1
fi

H=(-H "Content-Type: application/json" -H "X-Token: $TOKEN")

post() { curl -s -X POST "$BASE/api/v1$1" "${H[@]}" -d "$2"; }

echo "=== Registering devices ==="
post /device '{"device_id":"dev-ios-001","platform":"ios","os_version":"18.4","app_version":"2.1.0","locale":"en_US"}'
post /device '{"device_id":"dev-ios-002","platform":"ios","os_version":"17.6","app_version":"2.0.0","locale":"ja_JP"}'
post /device '{"device_id":"dev-android-001","platform":"android","os_version":"15","app_version":"2.0.3","locale":"de_DE"}'
post /device '{"device_id":"dev-android-002","platform":"android","os_version":"14","app_version":"1.9.0","locale":"es_ES"}'
post /device '{"device_id":"dev-web-001","platform":"web","os_version":"Chrome 130","app_version":"1.0.0","locale":"fr_FR"}'
post /device '{"device_id":"dev-macos-001","platform":"macos","os_version":"15.3","app_version":"2.1.0","locale":"en_GB"}'
echo ""
echo "6 devices registered"

echo ""
echo "=== Identifying users ==="
post /identify '{"device_id":"dev-ios-001","anonymous_id":"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"}'
post /identify '{"device_id":"dev-ios-002","anonymous_id":"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"}'
post /identify '{"device_id":"dev-android-001","anonymous_id":"b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3"}'
post /identify '{"device_id":"dev-web-001","anonymous_id":"c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"}'
echo ""
echo "4 users identified (ios-001 and ios-002 share same user)"

echo ""
echo "=== Sending events ==="

# Day 1: March 28
post /track '{
  "device_id":"dev-ios-001",
  "events":[
    {"id":"e001","name":"app_launch","timestamp":"2026-03-28T09:00:00+01:00","properties":{"source":{"type":"string","value":"organic"}}},
    {"id":"e002","name":"page_view","timestamp":"2026-03-28T09:01:00+01:00","properties":{"page":{"type":"string","value":"home"},"load_time":{"type":"float","value":0.342}}},
    {"id":"e003","name":"page_view","timestamp":"2026-03-28T09:05:00+01:00","properties":{"page":{"type":"string","value":"products"}}},
    {"id":"e004","name":"search","timestamp":"2026-03-28T09:06:00+01:00","properties":{"query":{"type":"string","value":"wireless headphones"},"results":{"type":"int","value":24}}},
    {"id":"e005","name":"product_view","timestamp":"2026-03-28T09:08:00+01:00","properties":{"product_id":{"type":"string","value":"SKU-1234"},"price":{"type":"float","value":79.99},"category":{"type":"string","value":"electronics"}}},
    {"id":"e006","name":"add_to_cart","timestamp":"2026-03-28T09:10:00+01:00","properties":{"product_id":{"type":"string","value":"SKU-1234"},"quantity":{"type":"int","value":1},"price":{"type":"float","value":79.99}}},
    {"id":"e007","name":"purchase","timestamp":"2026-03-28T09:15:00+01:00","properties":{"amount":{"type":"float","value":79.99},"currency":{"type":"string","value":"USD"},"items":{"type":"int","value":1},"is_first_purchase":{"type":"bool","value":true}}}
  ]
}'

post /track '{
  "device_id":"dev-android-001",
  "events":[
    {"id":"e010","name":"app_launch","timestamp":"2026-03-28T10:00:00+02:00","properties":{"source":{"type":"string","value":"push_notification"}}},
    {"id":"e011","name":"page_view","timestamp":"2026-03-28T10:01:00+02:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e012","name":"page_view","timestamp":"2026-03-28T10:03:00+02:00","properties":{"page":{"type":"string","value":"deals"}}},
    {"id":"e013","name":"product_view","timestamp":"2026-03-28T10:05:00+02:00","properties":{"product_id":{"type":"string","value":"SKU-5678"},"price":{"type":"float","value":29.99},"category":{"type":"string","value":"accessories"}}},
    {"id":"e014","name":"add_to_cart","timestamp":"2026-03-28T10:07:00+02:00","properties":{"product_id":{"type":"string","value":"SKU-5678"},"quantity":{"type":"int","value":2},"price":{"type":"float","value":29.99}}}
  ]
}'

post /track '{
  "device_id":"dev-web-001",
  "events":[
    {"id":"e020","name":"app_launch","timestamp":"2026-03-28T14:00:00+01:00","properties":{"source":{"type":"string","value":"direct"}}},
    {"id":"e021","name":"page_view","timestamp":"2026-03-28T14:01:00+01:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e022","name":"page_view","timestamp":"2026-03-28T14:02:00+01:00","properties":{"page":{"type":"string","value":"about"}}},
    {"id":"e023","name":"signup","timestamp":"2026-03-28T14:05:00+01:00","properties":{"method":{"type":"string","value":"email"}}}
  ]
}'

# Day 2: March 29
post /track '{
  "device_id":"dev-ios-001",
  "events":[
    {"id":"e030","name":"app_launch","timestamp":"2026-03-29T08:30:00+01:00","properties":{"source":{"type":"string","value":"organic"}}},
    {"id":"e031","name":"page_view","timestamp":"2026-03-29T08:31:00+01:00","properties":{"page":{"type":"string","value":"orders"}}},
    {"id":"e032","name":"page_view","timestamp":"2026-03-29T08:35:00+01:00","properties":{"page":{"type":"string","value":"home"}}}
  ]
}'

post /track '{
  "device_id":"dev-android-002",
  "events":[
    {"id":"e040","name":"app_launch","timestamp":"2026-03-29T12:00:00+02:00","properties":{"source":{"type":"string","value":"app_store"}}},
    {"id":"e041","name":"page_view","timestamp":"2026-03-29T12:01:00+02:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e042","name":"page_view","timestamp":"2026-03-29T12:03:00+02:00","properties":{"page":{"type":"string","value":"products"}}},
    {"id":"e043","name":"search","timestamp":"2026-03-29T12:04:00+02:00","properties":{"query":{"type":"string","value":"running shoes"},"results":{"type":"int","value":56}}},
    {"id":"e044","name":"product_view","timestamp":"2026-03-29T12:06:00+02:00","properties":{"product_id":{"type":"string","value":"SKU-9012"},"price":{"type":"float","value":129.99},"category":{"type":"string","value":"footwear"}}},
    {"id":"e045","name":"purchase","timestamp":"2026-03-29T12:15:00+02:00","properties":{"amount":{"type":"float","value":129.99},"currency":{"type":"string","value":"EUR"},"items":{"type":"int","value":1},"is_first_purchase":{"type":"bool","value":true}}}
  ]
}'

post /track '{
  "device_id":"dev-ios-002",
  "events":[
    {"id":"e050","name":"app_launch","timestamp":"2026-03-29T20:00:00+09:00","properties":{"source":{"type":"string","value":"organic"}}},
    {"id":"e051","name":"page_view","timestamp":"2026-03-29T20:01:00+09:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e052","name":"page_view","timestamp":"2026-03-29T20:05:00+09:00","properties":{"page":{"type":"string","value":"favorites"}}}
  ]
}'

# Day 3: March 30
post /track '{
  "device_id":"dev-macos-001",
  "events":[
    {"id":"e060","name":"app_launch","timestamp":"2026-03-30T11:00:00+00:00","properties":{"source":{"type":"string","value":"direct"}}},
    {"id":"e061","name":"page_view","timestamp":"2026-03-30T11:01:00+00:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e062","name":"page_view","timestamp":"2026-03-30T11:05:00+00:00","properties":{"page":{"type":"string","value":"settings"}}},
    {"id":"e063","name":"settings_change","timestamp":"2026-03-30T11:06:00+00:00","properties":{"setting":{"type":"string","value":"notifications"},"enabled":{"type":"bool","value":false}}}
  ]
}'

post /track '{
  "device_id":"dev-android-001",
  "events":[
    {"id":"e070","name":"app_launch","timestamp":"2026-03-30T15:00:00+02:00","properties":{"source":{"type":"string","value":"organic"}}},
    {"id":"e071","name":"page_view","timestamp":"2026-03-30T15:01:00+02:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e072","name":"purchase","timestamp":"2026-03-30T15:10:00+02:00","properties":{"amount":{"type":"float","value":59.98},"currency":{"type":"string","value":"EUR"},"items":{"type":"int","value":2},"is_first_purchase":{"type":"bool","value":false}}}
  ]
}'

# Day 4: March 31
post /track '{
  "device_id":"dev-web-001",
  "events":[
    {"id":"e080","name":"app_launch","timestamp":"2026-03-31T09:00:00+01:00","properties":{"source":{"type":"string","value":"social_media"}}},
    {"id":"e081","name":"page_view","timestamp":"2026-03-31T09:01:00+01:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e082","name":"page_view","timestamp":"2026-03-31T09:03:00+01:00","properties":{"page":{"type":"string","value":"products"}}},
    {"id":"e083","name":"search","timestamp":"2026-03-31T09:04:00+01:00","properties":{"query":{"type":"string","value":"laptop stand"},"results":{"type":"int","value":12}}},
    {"id":"e084","name":"product_view","timestamp":"2026-03-31T09:06:00+01:00","properties":{"product_id":{"type":"string","value":"SKU-3456"},"price":{"type":"float","value":49.99},"category":{"type":"string","value":"accessories"}}},
    {"id":"e085","name":"add_to_cart","timestamp":"2026-03-31T09:08:00+01:00","properties":{"product_id":{"type":"string","value":"SKU-3456"},"quantity":{"type":"int","value":1},"price":{"type":"float","value":49.99}}},
    {"id":"e086","name":"purchase","timestamp":"2026-03-31T09:15:00+01:00","properties":{"amount":{"type":"float","value":49.99},"currency":{"type":"string","value":"USD"},"items":{"type":"int","value":1},"is_first_purchase":{"type":"bool","value":false}}}
  ]
}'

post /track '{
  "device_id":"dev-ios-001",
  "events":[
    {"id":"e090","name":"app_launch","timestamp":"2026-03-31T17:00:00+01:00","properties":{"source":{"type":"string","value":"organic"}}},
    {"id":"e091","name":"page_view","timestamp":"2026-03-31T17:01:00+01:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e092","name":"page_view","timestamp":"2026-03-31T17:05:00+01:00","properties":{"page":{"type":"string","value":"products"}}},
    {"id":"e093","name":"product_view","timestamp":"2026-03-31T17:07:00+01:00","properties":{"product_id":{"type":"string","value":"SKU-7890"},"price":{"type":"float","value":199.99},"category":{"type":"string","value":"electronics"}}}
  ]
}'

# Day 5: April 1
post /track '{
  "device_id":"dev-android-002",
  "events":[
    {"id":"e100","name":"app_launch","timestamp":"2026-04-01T08:00:00+02:00","properties":{"source":{"type":"string","value":"organic"}}},
    {"id":"e101","name":"page_view","timestamp":"2026-04-01T08:01:00+02:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e102","name":"page_view","timestamp":"2026-04-01T08:03:00+02:00","properties":{"page":{"type":"string","value":"orders"}}},
    {"id":"e103","name":"app_rating","timestamp":"2026-04-01T08:10:00+02:00","properties":{"rating":{"type":"int","value":5},"comment":{"type":"string","value":"Great app!"}}}
  ]
}'

post /track '{
  "device_id":"dev-ios-002",
  "events":[
    {"id":"e110","name":"app_launch","timestamp":"2026-04-01T21:00:00+09:00","properties":{"source":{"type":"string","value":"organic"}}},
    {"id":"e111","name":"page_view","timestamp":"2026-04-01T21:01:00+09:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e112","name":"search","timestamp":"2026-04-01T21:03:00+09:00","properties":{"query":{"type":"string","value":"keyboard"},"results":{"type":"int","value":31}}},
    {"id":"e113","name":"product_view","timestamp":"2026-04-01T21:05:00+09:00","properties":{"product_id":{"type":"string","value":"SKU-4567"},"price":{"type":"float","value":159.99},"category":{"type":"string","value":"electronics"}}},
    {"id":"e114","name":"add_to_cart","timestamp":"2026-04-01T21:07:00+09:00","properties":{"product_id":{"type":"string","value":"SKU-4567"},"quantity":{"type":"int","value":1},"price":{"type":"float","value":159.99}}},
    {"id":"e115","name":"purchase","timestamp":"2026-04-01T21:15:00+09:00","properties":{"amount":{"type":"float","value":159.99},"currency":{"type":"string","value":"JPY"},"items":{"type":"int","value":1},"is_first_purchase":{"type":"bool","value":false}}}
  ]
}'

# Day 6: April 2
post /track '{
  "device_id":"dev-macos-001",
  "events":[
    {"id":"e120","name":"app_launch","timestamp":"2026-04-02T10:00:00+00:00","properties":{"source":{"type":"string","value":"direct"}}},
    {"id":"e121","name":"page_view","timestamp":"2026-04-02T10:01:00+00:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e122","name":"page_view","timestamp":"2026-04-02T10:05:00+00:00","properties":{"page":{"type":"string","value":"products"}}},
    {"id":"e123","name":"product_view","timestamp":"2026-04-02T10:07:00+00:00","properties":{"product_id":{"type":"string","value":"SKU-1234"},"price":{"type":"float","value":79.99},"category":{"type":"string","value":"electronics"}}},
    {"id":"e124","name":"purchase","timestamp":"2026-04-02T10:20:00+00:00","properties":{"amount":{"type":"float","value":79.99},"currency":{"type":"string","value":"GBP"},"items":{"type":"int","value":1},"is_first_purchase":{"type":"bool","value":true}}}
  ]
}'

post /track '{
  "device_id":"dev-ios-001",
  "events":[
    {"id":"e130","name":"app_launch","timestamp":"2026-04-02T13:00:00+01:00","properties":{"source":{"type":"string","value":"organic"}}},
    {"id":"e131","name":"page_view","timestamp":"2026-04-02T13:01:00+01:00","properties":{"page":{"type":"string","value":"home"}}},
    {"id":"e132","name":"error","timestamp":"2026-04-02T13:02:00+01:00","properties":{"code":{"type":"string","value":"NETWORK_TIMEOUT"},"screen":{"type":"string","value":"checkout"}}}
  ]
}'

echo ""
echo ""
echo "=== Seed complete ==="
echo "  6 devices"
echo "  4 identified users (3 unique)"
echo "  ~70 events across 6 days (Mar 28 - Apr 2)"
echo "  Event types: app_launch, page_view, search, product_view, add_to_cart, purchase, signup, settings_change, app_rating, error"
echo "  Properties: strings, ints, floats, booleans"
echo "  Currencies: USD, EUR, JPY, GBP"
