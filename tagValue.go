package main

import "errors"

var FreezeError = errors.New("tag must not be updated after population")

type TagValue struct {
	m       map[string]string
	fFreeze bool
}

func (receiver *TagValue) Put(key string, value string) error  {
	if receiver.fFreeze {
		return FreezeError
	}
	receiver.m[key] = value
	return nil
}

func (receiver *TagValue) Get(key string, defaultValue string) string  {
	if value, ok := receiver.m[key]; ok {
		return value
	}
	return defaultValue
}

func (receiver *TagValue) Has(key string) bool  {
	if _, ok := receiver.m[key]; ok {
		return true
	}
	return false
}

func (receiver *TagValue) Clear()  {
	for index, _ := range receiver.m {
		delete(receiver.m, index)
	}
	receiver.fFreeze = false
}

func (receiver *TagValue) freeze()  {
	receiver.fFreeze = true
}

func NewTagValue() (*TagValue, error)  {
	return &TagValue{
		m:       make(map[string]string),
		fFreeze: false,
	}, nil
}
