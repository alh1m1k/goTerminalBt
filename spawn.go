package main

import (
	"GoConsoleBT/collider"
	"errors"
	"fmt"
	lfpool "github.com/xiaonanln/go-lockfree-pool"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

var BuilderNotFoundError = errors.New("builder not found")

var PosAuto = Point{
	X: -math.MaxFloat64,
	Y: -math.MaxFloat64,
}

type Pooler interface {
	Put(x interface{})
	Get() interface{}
}

type Configurator func(object ObjectInterface, config interface{}) ObjectInterface

type SpawnManager struct {
	updater                      *Updater
	render                       Renderer
	animator                     *AnimationManager
	collider                     *collider.Collider
	location                     *Location
	config                       *GameConfig
	spawned                      map[ObjectInterface]bool
	builders                     map[string]Builder
	pendingSpawn, pendingDeSpawn []ObjectInterface
	respawn                      map[string]Pooler
	UnitEventChanel              EventChanel
	spawnMutex, deSpawnMutex     sync.Mutex
	planeDeSpawnAll              bool
	cycleSpawned                 int64 //only pooled
	cycleCreated                 int64 //only pooled
	Flags                        struct {
		lockFree bool
	}
}

func (manager *SpawnManager) Execute(timeLeft time.Duration) {
	var deSpawnAll bool

	manager.deSpawnMutex.Lock()

	deSpawnAll = manager.planeDeSpawnAll
	manager.planeDeSpawnAll = false

	if deSpawnAll {
		manager.pendingDeSpawn = manager.pendingDeSpawn[0:0]
		for object, spawned := range manager.spawned {
			if spawned {
				manager.pendingDeSpawn = append(manager.pendingDeSpawn, object)
			}
		}
	}

	for i, object := range manager.pendingDeSpawn {
		//logger.Println("despawn updater", object)
		manager.updater.Remove(object)
		manager.collider.Remove(object)
		manager.render.Remove(object)
		manager.pendingDeSpawn[i] = nil
		manager.spawned[object] = false
		object.DeSpawn()
		bl := object.GetBlueprint()
		if bl != "" && !deSpawnAll && manager.Flags.lockFree {
			if poll, ok := manager.respawn[bl]; ok {
				poll.Put(object)
			}
		}
		if DEBUG_SPAWN {
			logger.Printf("DeSpawn Object %CollisionInfo %+v \n", object, object)
		}
	}
	manager.pendingDeSpawn = manager.pendingDeSpawn[0:0]
	manager.deSpawnMutex.Unlock()

	manager.spawnMutex.Lock()

	if deSpawnAll {
		manager.pendingSpawn = manager.pendingSpawn[0:0]
	}

	for i, object := range manager.pendingSpawn {
		manager.updater.Add(object)
		manager.collider.Add(object)
		manager.render.Add(object)
		manager.pendingSpawn[i] = nil
		manager.spawned[object] = true
		object.Spawn()
		if DEBUG_SPAWN {
			logger.Printf("Spawn Object %CollisionInfo %+v \n", object, object)
		}
		if DEBUG {
			if manager.cycleSpawned > 0 {
				logger.Printf("spawned %d new %d \n", manager.cycleSpawned, manager.cycleCreated)
				manager.cycleSpawned = 0
				manager.cycleCreated = 0
			}
		}
	}
	manager.pendingSpawn = manager.pendingSpawn[0:0]
	manager.spawnMutex.Unlock()
}

func (manager *SpawnManager) Collect() {
	manager.deSpawnMutex.Lock()
	manager.spawnMutex.Lock()
	defer manager.spawnMutex.Unlock()
	defer manager.deSpawnMutex.Unlock()

	// that's effective than collect after despawn, probably pool depletion

	if manager.Flags.lockFree {
		return
	}

	for object, spawned := range manager.spawned {
		if !spawned {
			bl := object.GetBlueprint()
			if bl == "" {
				continue
			}
			if poll, ok := manager.respawn[bl]; ok {
				poll.Put(object)
			}
		}
	}
}

func (manager *SpawnManager) Spawn(coordinate Point, blueprint string, configurator Configurator, config interface{}) error {

	if _, ok := manager.builders[blueprint]; !ok {
		return fmt.Errorf("%s: %w", blueprint, BuilderNotFoundError)
	}

	candidate := manager.respawn[blueprint].Get()
	if candidate == nil {
		return errors.New("unable to create object")
	}
	object := candidate.(ObjectInterface)

	if configurator != nil {
		configurator(object, config)
	}

	if coordinate == PosAuto {
		if configurator != nil {
			//set by configurator
		} else {
			object.GetClBody().Move(0, 0)
		}
	} else {
		object.GetClBody().Move(coordinate.X, coordinate.Y)
	}

	object.Reset() //todo check animation and other then late reset

	manager.spawnMutex.Lock()
	manager.pendingSpawn = append(manager.pendingSpawn, object)
	manager.spawnMutex.Unlock()

	if DEBUG {
		atomic.AddInt64(&manager.cycleSpawned, 1)
	}

	return nil
}

func (manager *SpawnManager) SpawnPlayerTank2(coordinate Point, blueprint string, player *Player) error {
	if DEBUG_SPAWN {
		logger.Printf("spawn<user-item> attempt %s \n", blueprint)
	}
	return manager.Spawn(coordinate, blueprint, PlayerConfigurator, player)
}

func (manager *SpawnManager) SpawnProjectile2(coordinate Point, blueprint string, owner *Unit) error {
	//coordinate no sense
	if DEBUG_SPAWN {
		logger.Printf("spawn<user-item> attempt %s \n", blueprint)
	}
	return manager.Spawn(coordinate, blueprint, ProjectileConfigurator, owner)
}

func (manager *SpawnManager) SpawnExplosion2(coordinate Point, blueprint string, from ObjectInterface) error {
	if DEBUG_SPAWN {
		logger.Printf("spawn<user-item> attempt %s \n", blueprint)
	}
	return manager.Spawn(coordinate, blueprint, ExplosionConfigurator, from)
}

func (manager *SpawnManager) SpawnCollectable2(coordinate Point, blueprint string, from *Unit) error {
	if DEBUG_SPAWN {
		logger.Printf("spawn<user-item> attempt %s \n", blueprint)
	}
	return manager.Spawn(coordinate, blueprint, CollectableConfigurator, from)
}

func (manager *SpawnManager) DeSpawn(object ObjectInterface) {
	manager.deSpawnMutex.Lock()
	manager.pendingDeSpawn = append(manager.pendingDeSpawn, object)
	manager.deSpawnMutex.Unlock()
}

func (manager *SpawnManager) DeSpawnAll() {
	manager.planeDeSpawnAll = true
}

func (manager *SpawnManager) QuerySpawnedByTag(tag string) []ObjectInterface {
	manager.deSpawnMutex.Lock()
	manager.spawnMutex.Lock()
	defer manager.spawnMutex.Unlock()
	defer manager.deSpawnMutex.Unlock()
	var result []ObjectInterface
	for object, spawned := range manager.spawned {
		if spawned && object.HasTag(tag) {
			result = append(result, object)
		}
	}
	return result
}

func (manager *SpawnManager) QuerySpawnedByTagCount(tag string) int64 {
	manager.deSpawnMutex.Lock()
	manager.spawnMutex.Lock()
	defer manager.spawnMutex.Unlock()
	defer manager.deSpawnMutex.Unlock()
	var result int64
	for object, spawned := range manager.spawned {
		if spawned && object.HasTag(tag) {
			result++
		}
	}
	return result
}

func (manager *SpawnManager) AddBuilder(blueprint string, builder Builder) {
	manager.spawnMutex.Lock()
	defer manager.spawnMutex.Unlock()
	manager.builders[blueprint] = builder
	if _, ok := manager.respawn[blueprint]; !ok {
		manager.setupPool(blueprint, 250, manager.builders[blueprint])
	}
}

func (manager *SpawnManager) setupPool(blueprint string, size int, builder Builder) {
	if manager.Flags.lockFree {
		manager.respawn[blueprint] = lfpool.NewFastPool(size, builder)
	} else {
		manager.respawn[blueprint] = &sync.Pool{
			New: builder,
		}
	}
}

func NewSpawner(updater *Updater, render Renderer, collider *collider.Collider, location *Location, config *GameConfig) (*SpawnManager, error) {

	if location != nil {
		location.SetupZones(Point{
			X: 8,
			Y: 4,
		})
	}

	instance := &SpawnManager{
		updater:         updater,
		render:          render,
		animator:        nil,
		collider:        collider,
		location:        location,
		config:          config,
		spawned:         make(map[ObjectInterface]bool, 25),
		builders:        make(map[string]Builder, 5),
		pendingSpawn:    make([]ObjectInterface, 0, 25),
		pendingDeSpawn:  make([]ObjectInterface, 0, 25),
		respawn:         make(map[string]Pooler, 0),
		UnitEventChanel: make(EventChanel),
		spawnMutex:      sync.Mutex{},
		deSpawnMutex:    sync.Mutex{},
		planeDeSpawnAll: false,
	}

	instance.Flags.lockFree = gameConfig.LockfreePool

	return instance, nil
}
