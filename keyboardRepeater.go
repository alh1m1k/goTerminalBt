package main

import "github.com/eiannone/keyboard"

type KeyboardRepeater struct {
	origin      <-chan keyboard.KeyEvent
	subscribers []chan keyboard.KeyEvent
}

func (receiver *KeyboardRepeater) Subscribe() <-chan keyboard.KeyEvent {
	ch := make(chan keyboard.KeyEvent)
	receiver.subscribers = append(receiver.subscribers, ch)
	return ch
}

func (receiver *KeyboardRepeater) Unsubscribe(chanel <-chan keyboard.KeyEvent) {
	newSubscribers := make([]chan keyboard.KeyEvent, 0, maxInt(len(receiver.subscribers)-1, 1))
	for idx, candidate := range receiver.subscribers {
		if candidate == chanel {
			newSubscribers = append(newSubscribers, receiver.subscribers[:idx]...)
			newSubscribers = append(newSubscribers, receiver.subscribers[idx+1:]...)
			close(candidate)
		}
	}
	receiver.subscribers = newSubscribers
}

func NewKeyboardRepeater(origin <-chan keyboard.KeyEvent) (*KeyboardRepeater, error) {
	instance := &KeyboardRepeater{
		origin:      origin,
		subscribers: make([]chan keyboard.KeyEvent, 0, 1),
	}
	go repeatDispatcher(instance)
	return instance, nil
}

func repeatDispatcher(repeater *KeyboardRepeater) {
	for {
		select {
		case key, ok := <-repeater.origin:
			for _, subscriber := range repeater.subscribers {
				if !ok {
					close(subscriber)
				} else {
					subscriber <- key
				}
			}
			if !ok {
				return
			}
		}
	}
}
