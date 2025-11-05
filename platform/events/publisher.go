package events

// Publisher emits trigger execution jobs to Kafka.
type Publisher struct{}

// NewPublisher builds a stub publisher; provide the Kafka client later.
func NewPublisher() *Publisher {
	return &Publisher{}
}

// Publish will enqueue a trigger execution job once implemented.
func (p *Publisher) Publish(triggerID string) error {
	// TODO: implement Kafka publishing
	return nil
}
