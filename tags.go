package main

import (
	"errors"
)

var (
	TagNoValue *TagValue = &TagValue{
		m:       nil,
		fFreeze: true,
		noValue: true,
	}
	FreezeError = errors.New("tag must not be updated after population")
)

type Tagable interface {
	HasTag(tag string) bool
	GetTagValue(tag string, key string, defaultValue string) (string, error)
}

type TagValue struct {
	m       map[string]string
	fFreeze bool
	noValue bool
}

func (receiver *TagValue) Put(key string, value string) error {
	if receiver.fFreeze {
		return FreezeError
	}
	receiver.m[key] = value
	return nil
}

func (receiver *TagValue) Get(key string, defaultValue string) string {
	if value, ok := receiver.m[key]; ok {
		return value
	}
	return defaultValue
}

func (receiver *TagValue) Has(key string) bool {
	if _, ok := receiver.m[key]; ok {
		return true
	}
	return false
}

func (receiver *TagValue) Clear() {
	for index, _ := range receiver.m {
		delete(receiver.m, index)
	}
	receiver.fFreeze = false
}

func (receiver *TagValue) freeze() {
	receiver.fFreeze = true
}

func NewTagValue() (*TagValue, error) {
	return &TagValue{
		m:       make(map[string]string),
		fFreeze: false,
	}, nil
}

type Tags struct {
	tagValues map[string]*TagValue
}

func (receiver *Tags) addTag(tags ...string) {
	for _, tag := range tags {
		receiver.tagValues[tag], _ = NewTagValue()
	}
}

func (receiver *Tags) HasTag(tag string) bool {
	if _, ok := receiver.tagValues[tag]; ok {
		return true
	}
	return false
}

func (receiver *Tags) clearTags() {
	for index, _ := range receiver.tagValues {
		delete(receiver.tagValues, index)
	}
}

func (receiver *Tags) removeTag(tag string) {
	delete(receiver.tagValues, tag)
}

func (receiver *Tags) GetTag(tag string) (*TagValue, error) {
	if tagValue, ok := receiver.tagValues[tag]; !ok {
		return nil, Tag404Error
	} else if tagValue.noValue {
		return nil, Tag404Error
	}
	return receiver.tagValues[tag], nil
}

func (receiver *Tags) GetTagValue(tag string, key string, defaultValue string) (string, error) {
	if tag, ok := receiver.tagValues[tag]; ok {
		return tag.Get(key, defaultValue), nil
	}
	return defaultValue, Tag404Error
}

func (receiver *Tags) Len() int {
	return len(receiver.tagValues)
}

func (receiver *Tags) Copy() *Tags {
	instance := *receiver
	instance.tagValues = make(map[string]*TagValue, len(receiver.tagValues))
	for index, value := range receiver.tagValues {
		instance.tagValues[index] = value
	}
	return &instance
}

func NewTags() (*Tags, error) {
	return &Tags{map[string]*TagValue{}}, nil
}
