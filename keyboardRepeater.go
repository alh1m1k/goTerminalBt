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
