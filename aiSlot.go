package main

var ()

type AiSlots struct {
	*Unit
	subscribers []IndexTracker
}

func (receiver *AiSlots) HasTag(value string) bool {
	return true
}

func (receiver *AiSlots) indexUpdate() {
	//todo make simpler
	for _, subscriber := range receiver.subscribers {
		if subscriber == nil {
			continue
		}
		//subscriber.OnIndexUpdate(receiver)
	}
}

func (receiver *AiSlots) Subscribe(subscriber IndexTracker) {
	receiver.subscribers = append(receiver.subscribers, subscriber)
}

func (receiver *AiSlots) Unsubscribe(subscriber IndexTracker) {
	for index, candidate := range receiver.subscribers {
		if subscriber == candidate {
			receiver.subscribers[index] = nil
		}
	}
}
