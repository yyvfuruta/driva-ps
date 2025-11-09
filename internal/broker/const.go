package broker

const (
	OrderEventsExchangeName = "order.events"
	OrderEventsExchangeType = "direct"

	// API sends to and Worker1 reads from:
	OrderCreatedQueue      = "order.created"
	OrderCreatedRoutingKey = "order.created"

	// Worker1 sends to and Worker2 reads from:
	EnrichmentRequestQueue      = "order.enrichment.requested"
	EnrichmentRequestRoutingKey = "order.enrichment.requested"

	// Worker2 sends to and Worker3 reads from:
	OrderEnrichedQueue      = "order.enriched"
	OrderEnrichedRoutingKey = "order.enriched"
)
