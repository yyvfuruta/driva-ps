#!/bin/sh

IDEMPOTENCY_KEY="$(uuid)"

curl -XPOST --location 'http://localhost:8080/orders' \
	-H 'Content-Type: application/json' \
	-H "X-Idempotency-Key: $IDEMPOTENCY_KEY" \
	-H 'Authorization: Bearer cecd24ae-bbd6-11f0-bbc0-2c0da77e0ef5' \
	--data '{
		"customer_id": "123",
		"total_amount": 15.21,
		"items": [
			{
				"sku": "ABC",
				"qty": 2
			},
			{
				"sku": "XYZ",
				"qty": 1
			}
		]
	}'

