package main

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

var StateNotFoundError = errors.New("state not found")
var StateExistError = errors.New("state exist in path")
var SameStateError = errors.New("state is same")
var NoOwnerError = errors.New("no owner to apply state")
var states map[string]*State = make(map[string]*State, 20)

var ToDefaultState = "{{DEFAULT_STATE}}"

type Stater interface {
	Enter(path string) error
	ApplyState(current *StateItem) error
}

type SateInfoBuilder func(map[string]interface{}) (interface{}, error)

type State struct {
	Owner             Stater
	root, Current     *StateItem
	path, defaultPath string
	mutex             sync.Mutex
}

//WARNING! apply lock on owner.ApplyState too
func (receiver *State) Enter(path string) error {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	if path == ToDefaultState {
		path = receiver.defaultPath //todo remove
	}
	if DEBUG_STATE {
		logger.Printf("attempt to enter state by path %s current path %s obj %T %+v", path, receiver.path, receiver.Owner, receiver.Owner)
	}
	newState, newPath, err := receiver.find(path)
	if err != nil {
		return err
	}
	if receiver.Current == newState {
		if DEBUG_STATE {
			logger.Printf("rejected to enter same state")
		}
		return SameStateError
	}

	err = receiver.ApplyState(newState)
	if err != nil {
		return err
	}

	receiver.Current = newState
	receiver.path = newPath

	if DEBUG_STATE {
		logger.Printf("new path is %s state %T", newPath, newState)
	}

	return nil
}

func (receiver *State) Reset() error {
	if DEBUG_STATE {
		logger.Printf("attempt to reset state")
	}
	receiver.Current = receiver.root //in case if defaultPath relative
	receiver.path = "/"
	return receiver.Enter(receiver.defaultPath)
}

func (receiver *State) MoveTo(path string) error {
	if DEBUG_STATE {
		logger.Printf("attempt to move to state %s", path)
	}
	newState, newPath, err := receiver.find(path)
	if err != nil {
		return err
	}
	receiver.Current = newState
	receiver.path = newPath
	return nil
}

func (receiver *State) find(path string) (*StateItem, string, error) {
	if DEBUG_STATE {
		logger.Printf("searching state %s", path)
	}
	var newState *StateItem
	var newPath string = receiver.path
	var err error
	if path == "/" {
		if DEBUG_STATE {
			logger.Printf("fast forward to root %s", path)
		}
		return receiver.root, "/", err
	}
	if receiver.isPathAbsolute(path) {
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		newState, err = receiver.findNextState(receiver.root, parts)
		newPath = path
	} else {
		parts := strings.Split(path, "/")
		newState, err = receiver.findNextState(receiver.Current, parts)
		if strings.HasSuffix(newPath, "/") {
			newPath = newPath + path
		} else {
			newPath = newPath + "/" + path
		}
	}
	if err != nil && !strings.Contains(path, "/") {
		newState, newPath, err = receiver.backwardNextState(receiver.Current, path)
	}
	if err != nil {
		return nil, "", err
	}
	return newState, newPath, err
}

func (receiver *State) Error() error {
	return nil
}

func (receiver *State) ApplyState(current *StateItem) error {
	if receiver.Owner != nil {
		return receiver.Owner.ApplyState(current)
	}
	return NoOwnerError
}

func (receiver *State) isPathAbsolute(path string) bool {
	return strings.HasPrefix(path, "/")
}

//non recursive
func (receiver *State) CreateState(absolutePath string, payload interface{}) (*StateItem, error) {
	absolutePath = strings.TrimPrefix(absolutePath, "/")
	parts := strings.Split(absolutePath, "/")
	place := parts[len(parts)-1]
	parts = parts[0 : len(parts)-1]
	parent, _, err := receiver.find("/" + strings.Join(parts, "/")) // empty path protection
	if err != nil {
		return nil, err
	}
	if _, ok := parent.items[place]; ok {
		return nil, StateExistError
	}
	newState, _ := NewStateItem(parent, payload)
	if parent.items == nil {
		parent.items = make(map[string]*StateItem)
	}
	parent.items[place] = newState
	return newState, nil
}

func (receiver *State) findNextState(start *StateItem, path []string) (*StateItem, error) {
	candidate := start
	if DEBUG_STATE {
		logger.Printf("findNextState attempt")
	}
	for _, item := range path {
		if state, ok := candidate.items[item]; ok {
			candidate = state
		} else {
			if DEBUG_STATE {
				logger.Printf("findNextState attempt failed")
			}
			return nil, StateNotFoundError
		}
	}
	return candidate, nil
}

func (receiver *State) backwardNextState(start *StateItem, path string) (state *StateItem, newPath string, err error) {
	candidate := start
	if DEBUG_STATE {
		logger.Printf("backwardNextState attempt %s", path)
	}
	if state, ok := candidate.items[path]; ok {
		logger.Printf("backwardNextState fast forward to child %s", path)
		return state, receiver.path + path, nil
	}
	currentPathParts := strings.Split(strings.TrimPrefix(receiver.path, "/"), "/")
	if candidate.parent != nil {
		if state, ok := candidate.parent.items[path]; ok {
			//ninja fix top->left
			currentPathParts[len(currentPathParts)-1] = path
			if DEBUG_STATE {
				logger.Printf("backwardNextState (neighbour) find %s", "/"+strings.Join(currentPathParts, "/"))
			}
			return state, "/" + strings.Join(currentPathParts, "/"), nil
		}
	}
	var backwardPath []string
	startIndex := len(currentPathParts)
	for candidate != nil && startIndex >= 0 {
		backwardPath = currentPathParts[startIndex:]

		subCandidate, err := receiver.doBackwardFind(candidate, backwardPath, path)

		if err == nil && subCandidate != nil {
			pathParts := make([]string, 0, len(currentPathParts)-len(backwardPath)+1)
			pathParts = append(pathParts, currentPathParts[0:len(currentPathParts)-len(backwardPath)]...)
			pathParts = append(pathParts, path)
			pathParts = append(pathParts, backwardPath[1:]...)
			return subCandidate, "/" + strings.Join(pathParts, "/"), nil
		} else {
			if DEBUG_STATE {
				logger.Printf("backwardNextState (backward) faild %s", path)
			}
		}
		startIndex--
		candidate = candidate.parent
	}

	return nil, "", StateNotFoundError
}

func (receiver *State) doBackwardFind(candidate *StateItem, backwardPath []string, path string) (*StateItem, error) {
	if subCandidate, ok := candidate.items[path]; ok {
		for i, item := range backwardPath {
			if i == 0 {
				continue
			}
			if DEBUG_STATE {
				logger.Printf("backwardNextState search in (backward) %s backward:", path)
			}
			if state, ok := subCandidate.items[item]; ok {
				if DEBUG_STATE {
					logger.Printf("backwardNextState search in (upward) %s :", item)
				}
				subCandidate = state
			} else {
				return nil, StateNotFoundError
			}
		}
		return subCandidate, nil
	}
	return nil, StateNotFoundError
}

func (receiver *State) Free() {

}

func (receiver *State) Copy() *State {
	instance := *receiver
	instance.mutex = sync.Mutex{}
	instance.root, instance.Current = instance.root.copy(instance.Current)
	return &instance
}

type StateItem struct {
	parent    *StateItem
	items     map[string]*StateItem
	StateInfo interface{}
}

func (receiver *StateItem) Copy() *StateItem {
	instance := *receiver
	instance.items = make(map[string]*StateItem, len(receiver.items))
	if receiver.StateInfo != nil {
		if info, ok := receiver.StateInfo.(*UnitStateInfo); ok {
			copy := *info
			copy.sprite = CopySprite(copy.sprite)
			instance.StateInfo = &copy
		}
	}
	for key, item := range receiver.items {
		instance.items[key] = item.Copy()
		instance.items[key].parent = &instance
	}
	return &instance
}

/**
return copy, updatedRef
*/
func (receiver *StateItem) copy(updateRefFor *StateItem) (*StateItem, *StateItem) {
	instance := *receiver
	instance.items = make(map[string]*StateItem, len(receiver.items))
	if receiver.StateInfo != nil {
		if info, ok := receiver.StateInfo.(*UnitStateInfo); ok {
			copy := *info
			copy.sprite = CopySprite(copy.sprite)
			instance.StateInfo = &copy
		}
	}
	for key, item := range receiver.items {
		instance.items[key], updateRefFor = item.copy(updateRefFor)
		instance.items[key].parent = &instance
	}
	if updateRefFor == receiver {
		updateRefFor = &instance
	}
	return &instance, updateRefFor
}

type stateRead struct {
	Default string
	Items   map[string]interface{}
}

func NewState(owner Stater) (*State, error) {
	root := StateItem{
		parent:    nil,
		items:     nil,
		StateInfo: nil,
	}
	state := State{
		Owner:   owner,
		root:    &root,
		Current: &root,
		path:    "/",
	}
	return &state, nil
}

func NewStateItem(owner *StateItem, payload interface{}) (*StateItem, error) {
	return &StateItem{
		parent:    owner,
		items:     make(map[string]*StateItem),
		StateInfo: payload,
	}, nil
}

func GetState(id string, builder SateInfoBuilder) (*State, error) {
	if state, ok := states[id]; ok {
		return state.Copy(), nil
	}
	buffer, err := loadState(id)
	if err != nil {
		return nil, err
	}
	stateRead := stateRead{}
	err = json.Unmarshal(buffer, &stateRead)
	if err != nil {
		return nil, err
	}
	if len(stateRead.Items) == 0 {
		return nil, errors.New("load empty state")
	}

	state, _ := NewState(nil)
	root := state.root

	recursiveCreateState(root, stateRead.Items, builder)

	if stateRead.Default != "" {
		err = state.MoveTo(stateRead.Default)
		if err != nil {
			return nil, err
		}
		state.defaultPath = stateRead.Default
	}

	states[id] = state
	return state.Copy(), nil
}

func recursiveCreateState(state *StateItem, scheme map[string]interface{}, builder SateInfoBuilder) {
	for index, mp := range scheme {
		item := StateItem{
			parent:    state,
			items:     nil,
			StateInfo: nil,
		}
		if state.items == nil {
			state.items = make(map[string]*StateItem)
		}
		state.items[index] = &item
		for index2, scheme2 := range mp.(map[string]interface{}) {
			if index2 == "items" {
				recursiveCreateState(&item, scheme2.(map[string]interface{}), builder)
			}
		}
		info := mp.(map[string]interface{})
		delete(info, "items")
		if builder != nil {
			info, _ := builder(info)
			item.StateInfo = info
		}
	}
}
