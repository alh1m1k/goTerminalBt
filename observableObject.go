package main


type Event struct {
	EType   int
	Object  interface{}
	Payload interface{}
}

type EventChanel chan Event

type ObservableObjectInterface interface {
	Trigger(event Event, object interface{}, payload interface{})
	GetEventChanel() EventChanel
}

type ObservableObject struct {
	Owner  ObservableObjectInterface
	output EventChanel
}

func (receiver *ObservableObject) Trigger(event Event, object interface{}, payload interface{}) {
	event.Object = object
	event.Payload = payload
	if DEBUG_EVENT {
		logger.Printf("trigger event %d, %T, %+v \n", event.EType, object, object)
	}
	receiver.output <- event
}

func (receiver *ObservableObject) GetEventChanel() EventChanel {
	return receiver.output
}

func (receiver *ObservableObject) Free() error {
	//close(receiver.output)
	return nil
}

func (receiver *ObservableObject) Copy() *ObservableObject {
	instance := *receiver
	return &instance
}

func NewObservableObject(output EventChanel, owner ObservableObjectInterface) (*ObservableObject, error) {
	instance := new(ObservableObject)
	instance.Owner = owner
	instance.output = output

	return instance, nil
}

