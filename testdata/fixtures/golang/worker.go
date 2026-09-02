package worker

// expected-load: monthly_calls=50_000 requests_per_active_minute=3
func Work() {}

// expected-load:
//
//	monthly_calls: 8_000
//	avg_conversation_turns: 4
//	confidence: low
func Chat() {}
