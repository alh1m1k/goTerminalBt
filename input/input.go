package main

import (
	"github.com/eiannone/keyboard"
)

func New()(<-chan keyboard.KeyEvent, error) {
	keysEvents, err := keyboard.GetKeys(10)
	return keysEvents, err
/*	if err != nil {
		panic(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	fmt.Println("Press ESC to quit")
	for {
		event := <-keysEvents
		if event.Err != nil {
			panic(event.Err)
		}
		fmt.Printf("You pressed: rune %q, key %X\r\n", event.Rune, event.Key)
		if event.Key == keyboard.KeyEsc {
			break
		}
	}*/
}
