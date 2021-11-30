package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

var LoaderNotFoundError = errors.New("loader not found")
var ParseError 			= errors.New("invalid json value")
var PrototypeError 		= errors.New("unable to copy from prototype")
var InstanceError 		= errors.New("unable to instance object")

type loadError struct {
	path   string
	err    error
}

func (receiver loadError) Error() string  {
	return fmt.Sprintf("\"%s\" at path %s", receiver.err, receiver.path)
}

func (receiver loadError) Unwrap() error {
	return receiver.err
}

type LoadErrors struct {
	e []error
	t []string //trace
}

func (receiver *LoadErrors) HasError() bool  {
	if len(receiver.e) > 0 {
		return true
	} else {
		return false
	}
}

func (receiver *LoadErrors) Error() string  {
	var s []string
	for _, err := range receiver.e {
		s = append(s, err.Error()) // Конвертирует ошибки в строки
	}
	return strings.Join(s, ", ")
}

//add and check error non null
func (receiver *LoadErrors) Add(error error)  bool  {
	if error != nil {
		var sl []string
		if len(receiver.t) > 1 {
			sl = receiver.t[1:] //without root
		} else {
			sl = receiver.t[:]
		}
		le := &loadError{
			path: "/" + strings.Join(sl, "/"),
			err:  error,
		}
		receiver.e = append(receiver.e, le)
		return true
	}
	return false
}

func (receiver *LoadErrors) tracePush(p string)  {
	receiver.t = append(receiver.t, p)
}

func (receiver *LoadErrors) tracePop() string {
	v := receiver.t[len(receiver.t)-1]
	receiver.t = receiver.t[0:len(receiver.t)-1]
	return v
}

func newLoadErrors() (*LoadErrors, error) {
	instance   := new(LoadErrors)
	instance.e	= make([]error, 0)
	instance.t 	= make([]string, 0)
	return instance, nil
}

type ObjectGetter 	func(blueprint string) interface{} //todo simplify loader by this getter ie encapsulate loader acquire and it's call
type LoaderGetter 	func(blueprint string) Loader
type Loader 		func(get LoaderGetter, eCollector *LoadErrors,  payload []byte) interface{}
type Builder 		func() interface{}

type FileBuf struct{
	buf []byte
	err error
}

type BlueprintManager struct {
	EventChanel   EventChanel
	GameConfig    *GameConfig
	FilePool      map[string]*FileBuf
	FilePath      string
	FileExtension string
	loaders       map[string] Loader
	proto         map[string] ObjectInterface
	protoShadow   map[string] ObjectInterface
	m             sync.Mutex
}

func (receiver *BlueprintManager) Get(blueprint string) (ObjectInterface, error)  {
	if object, ok := receiver.proto[blueprint]; ok {
		return receiver.copy(object)
	}
	receiver.m.Lock()
	defer receiver.m.Unlock() //concurent sprite and collector problem
	if object, ok := receiver.proto[blueprint]; ok {
		return receiver.copy(object) //try again
	}
	payload, _ := receiver.load(blueprint)
	if root, ok := receiver.loaders["/"]; ok {
		collector, _ := newLoadErrors()
		if stuff := root(receiver.getLoader, collector, payload); stuff == nil {
			collector.Add(errors.New("object wont created"))
			return nil, collector
		} else {
			if object, ok := stuff.(ObjectInterface); ok {
				object = receiver.postProcess(object, collector)

				//add object to map without lock
/*				receiver.protoShadow[blueprint] = object
				pShadow := unsafe.Pointer(&receiver.protoShadow)
				pProto 	:= unsafe.Pointer(&receiver.proto)
				pShadow  = atomic.SwapPointer(&pProto, pShadow)
				receiver.proto = *(*map[string] ObjectInterface)(pProto)
				receiver.protoShadow = *(*map[string] ObjectInterface)(pShadow)*/

				receiver.protoShadow[blueprint] = object
				receiver.proto, receiver.protoShadow = receiver.protoShadow, receiver.proto
				if _, ok := receiver.protoShadow[blueprint]; ok {
					panic("shadow copy violation")
				} else {
					receiver.protoShadow[blueprint] = object
				}

				object, err := receiver.copy(object)
				collector.Add(err)
				if !collector.HasError() {
					return object, nil
				} else {
					return object, collector
				}
			} else {
				return nil, collector
			}
		}
	}
	return nil, LoaderNotFoundError
}

func (receiver *BlueprintManager) CreateBuilder(blueprint string) (Builder, error)  {
	object, err := receiver.Get(blueprint)
	if object == nil { //may be error even on success
		if err != nil {
			logger.Println(err.Error())
		}
		return nil, err
	}
	if err != nil {
		logger.Println(err.Error()) //try to ignore spam of errors, logs only first
	}

	return func() interface{} {
		obj, _ := receiver.Get(blueprint) //todo do some, log errors
		return obj
	}, nil
}

func (receiver *BlueprintManager) AddLoader(blueprint string, loader Loader) {
	receiver.loaders[blueprint] = receiver.wrapLoader(loader, blueprint)
}

func (receiver *BlueprintManager) AddLoaderPackage(p *Package) {
	for blueprint, loader := range p.M {
		receiver.loaders[blueprint] = receiver.wrapLoader(loader, blueprint)
	}
	receiver.FilePath 		= p.FilePath
	receiver.FileExtension 	= p.FileExtension
}

func (receiver *BlueprintManager) wrapLoader(loader Loader, blueprint string) Loader  {
	return func(get LoaderGetter, eCollector *LoadErrors, payload []byte) interface{} {
		eCollector.tracePush(blueprint) //trace wrapper
		ret := loader(get, eCollector, payload)
		eCollector.tracePop()
		return ret
	}
}

func (receiver *BlueprintManager) postProcess(object ObjectInterface, collector *LoadErrors) ObjectInterface  {
	if object == nil {
		return nil
	}
	if object.GetAttr().Collided {
		w, h := object.GetClBody().GetWH()
		object.GetClBody().Resize(w * receiver.GameConfig.ColWidth, h * receiver.GameConfig.RowHeight)
	}
	if object.GetAttr().Motioner { //todo simplify //todo move to render
		proxy := object.(Motioner)
		scale := 2.0 //receiver.GameConfig.ColWidth / receiver.GameConfig.RowHeight
		//pixel at Y aprox twice bigger than X
		proxy.GetSpeed().Y = proxy.GetSpeed().X / scale
		acl := object.(Accelerator)
		acl.GetMaxSpeed().Y = acl.GetMaxSpeed().X / scale
		acl.GetMinSpeed().Y = acl.GetMinSpeed().X / scale
	}
	return object
}

func (receiver *BlueprintManager) load(id string) ([]byte, error) {
	if content, ok := receiver.FilePool[id]; !ok {
		buf, err := os.ReadFile(receiver.FilePath + "/" + id + "." +  receiver.FileExtension)
		receiver.FilePool[id] = &FileBuf{
			buf: buf,
			err: err,
		}
		return receiver.FilePool[id].buf, receiver.FilePool[id].err
	} else {
		return content.buf, content.err
	}
}

func (receiver *BlueprintManager) getLoader(blueprint string) Loader {
	return receiver.loaders[blueprint]
}

func (receiver *BlueprintManager) copy(object ObjectInterface) (ObjectInterface, error) {
	switch object.(type) {
	case *Unit:
		unit := object.(*Unit).Copy()
		unit.Prototype = object
		return unit, nil
	case *Wall:
		wall := object.(*Wall).Copy()
		wall.Prototype = object
		return wall, nil
	case *Projectile:
		projectile := object.(*Projectile).Copy()
		projectile.Prototype = object
		return projectile, nil
	case *Explosion:
		explosion := object.(*Explosion).Copy()
		explosion.Prototype = object
		return explosion, nil
	case *Collectable:
		collectable := object.(*Collectable).Copy()
		collectable.Prototype = object
		return collectable, nil
	}
	return nil, PrototypeError
}

func NewBlueprintManager() (*BlueprintManager, error)  {
	instance := new(BlueprintManager)

	instance.FilePool 	= make(map[string] *FileBuf)
	instance.loaders 	= make(map[string] Loader)
	instance.proto 			= make(map[string] ObjectInterface)
	instance.protoShadow 	= make(map[string] ObjectInterface)

	instance.AddLoader("eventChanel", func(get LoaderGetter, eCollector *LoadErrors, payload []byte) interface{} {
		return instance.EventChanel
	})
	instance.AddLoader("gameConfig", func(get LoaderGetter, eCollector *LoadErrors, payload []byte) interface{} {
		return instance.GameConfig
	})

	return instance, nil
}


