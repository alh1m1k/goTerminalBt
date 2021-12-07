package main

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const GAME_START = 200
const GAME_END_WIN = 201
const GAME_END_LOSE = 202

var GameInProgress = errors.New("game in progress")

type gameActionCallback func(game Game, object ObjectInterface, payload interface{}) error

type GameAction struct {
	callback  gameActionCallback
	object    ObjectInterface
	payload   interface{}
	waitUnitl time.Time
}

type Game struct {
	players []*Player
	*SpawnManager
	*ObservableObject
	*Location
	*EffectManager
	scenario                 *Scenario
	terminator               chan bool
	spawnedPlayer, spawnedAi int64
	inProgress               bool
	mutex, delay             sync.Mutex
	delayedAction            []*GameAction
	nextDelayedTaskExec      time.Time
	delayedTaskChan          <-chan time.Time
}

func (receiver *Game) AddPlayer(player *Player) error {
	if receiver.inProgress {
		return GameInProgress
	}
	receiver.players = append(receiver.players, player)
	return nil
}

func (receiver *Game) GetPlayers() []*Player {
	return receiver.players
}

func (receiver *Game) Run(scenario *Scenario) error {

	receiver.mutex.Lock()
	if receiver.inProgress || scenario == nil {
		receiver.mutex.Unlock()
		return GameInProgress
	}
	receiver.inProgress = true
	receiver.mutex.Unlock()

	receiver.scenario = scenario
	receiver.spawnedPlayer = 0
	receiver.terminator = make(chan bool)

	go scenarioDispatcher(game, scenario.GetEventChanel(), receiver.terminator)
	go gameCmdDispatcher(game, game.SpawnManager.UnitEventChanel, receiver.terminator)

	err := scenario.Enter("start")

	if err != nil {
		close(receiver.terminator)
		receiver.scenario = nil
		receiver.SpawnManager.DeSpawnAll()
		receiver.inProgress = false
		logger.Print(err)
		return err
	}

	for _, player := range receiver.players {
		location, _ := receiver.location.Coordinate2Spawn(true)
		err := receiver.SpawnManager.SpawnPlayerTank2(location, "player-tank", player)
		if err != nil {
			logger.Println("at spawning player error: ", err)
		}
		receiver.spawnedPlayer++
	}

	receiver.Trigger(Event{
		EType:   GAME_START,
		Object:  nil,
		Payload: nil,
	}, receiver, nil)

	return nil
}

func (receiver *Game) onSpawnRequest(scenario *Scenario, payload *SpawnRequest) {
	var location Point
	if payload.Location == PosAuto {
		if receiver.Location != nil {
			location, _ = receiver.Location.Coordinate2Spawn(true)
		} else {
			location = Point{}
		}
	} else {
		location = payload.Location
	}
	receiver.SpawnManager.Spawn(location, payload.Blueprint, DefaultConfigurator, payload)
}

func (receiver *Game) onUnitFire(object *Unit, payload interface{}) {
	if object.Gun != nil && object.Gun.GetProjectile() != "" {
		err := receiver.SpawnManager.SpawnProjectile2(PosAuto, object.Gun.GetProjectile(), object)
		if err != nil {
			logger.Printf("unable to fire %s due: %s \n", object.Gun.GetProjectile(), err)
		}
	} else {
		logger.Printf("unable to fire due projectile or gun not found %s \n", object.Gun.GetProjectile())
	}
}

func (receiver *Game) onUnitDamage(object ObjectInterface, payload interface{}) {
	if object.HasTag("wall") {
		wall := object.(*Wall)
		wallHp := int(math.Max(float64(wall.HP), 1))
		if wall.FullHP/wallHp >= 2 {
			wall.Enter("damage")
		}
	}
	if object.HasTag("highlights-damage") {
		toState, _ := object.GetTagValue("highlights-damage", "moveToState", "receiveDamage")
		returnToState, _ := object.GetTagValue("highlights-damage", "returnToState", ToDefaultState)
		object.(Stater).Enter(toState)
		delayedEnterState(object.(Stater), returnToState, time.Millisecond*500)
	}
	if object.HasTag("tank") {
		tank := object.(*Unit)
		tankHp := int(math.Max(float64(tank.HP), 1))
		if tank.FullHP/tankHp > 2 {
			//wall.Enter("damage") //todo
		}
	}
}

func (receiver *Game) onObjectReset(object ObjectInterface, payload interface{}) {
	/*	if object.HasTag("highlights-appear") {
		object.(Stater).Enter("appear")
		delayedEnterState(object.(Stater), "normal", object.(Appearable).GetAppearDuration())
	}*/
}

func (receiver *Game) onObjectSpawn(object ObjectInterface, payload interface{}) {
	if object.HasTag("highlights-appear") {
		object.(Stater).Enter("appear")
		delayedEnterState(object.(Stater), "normal", object.(Appearable).GetAppearDuration())
	}
}

func (receiver *Game) onObjectDestroy(object ObjectInterface, payload interface{}) {
	var despawnNow = true

	if object.HasTag("scored") && payload != nil {
		nemesis := payload.(ObjectInterface)
		if nemesis.HasTag("player") {
			player := receiver.playerByUnit(nemesis)
			if player != nil {
				player.IncScore(int64(object.(Scored).GetScore()))
			}
		}
	}

	if object.HasTag("explosive") {
		bl, _ := object.GetTagValue("explosive", "blueprint", "tank-base-explosion")
		err := receiver.SpawnManager.SpawnExplosion2(PosAuto, bl, object)
		if err != nil {
			logger.Printf("unable to spawn explosion: %s \n", err)
		}
		receiver.EffectManager.applyGlobalShake(0.3, time.Second*1)
	}

	if object.HasTag("fanout") {
		doFanoutSpawn(receiver, object)
	}

	if object.HasTag("highlights-disappear") {
		despawnNow = false
		object.(Stater).Enter("disappear")
		time.AfterFunc(object.(Disappearable).GetDisappearDuration(), func() {
			receiver.SpawnManager.DeSpawn(object)
		})
	}

	if despawnNow {
		receiver.SpawnManager.DeSpawn(object)
	}
}

func (receiver *Game) onUnitCollect(object *Collectable, payload interface{}) {
	//make Geschäft
	unit := payload.(*Unit)
	player := receiver.playerByUnit(unit)

	if object.HasTag("opel") {
		if player != nil {
			player.IncScore(int64((rand.Intn(2) - 1) * rand.Intn(1000)))
		}
		if rand.Intn(4) > 3 {
			unit.Gun.Downgrade()
		} else {
			unit.Gun.IncAmmoIfAcceptable(1)
		}
	}
	if object.HasTag("gun") {
		seed := rand.Intn(10)
		if seed <= 3 {
			unit.Gun.Current.ShotQueue += 2
		}
		if seed <= 8 && seed > 3 {

			unit.Gun.Upgrade(&GunState{
				Projectile:       GetConventionalProjectileName(),
				Ammo:             10,
				ShotQueue:        1,
				PerShotQueueTime: time.Second / 5,
				ReloadTime:       2 * time.Second,
			})

		}
		if seed >= 9 {

			unit.Gun.Upgrade(&GunState{
				Projectile:       "tank-base-projectile-apocalypse",
				Ammo:             1,
				ShotQueue:        1,
				PerShotQueueTime: time.Second / 2,
				ReloadTime:       5 * time.Second,
			})

		}
		unit.Gun.IncAmmoIfAcceptable(2)
		if player != nil {
			player.IncScore(100)
		}
	}

}

func (receiver *Game) onObjectDeSpawn(object ObjectInterface, payload interface{}) {
	if object.HasTag("player") {
		for _, player := range receiver.players {
			if player.Unit == object {
				left := player.DecrRetry(1)
				if left <= 0 {
					if atomic.AddInt64(&receiver.spawnedPlayer, -1) == 0 {
						receiver.End(GAME_END_LOSE)
					}
				} else {
					location, _ := receiver.location.Coordinate2Spawn(true)
					receiver.SpawnManager.SpawnPlayerTank2(location, "player-tank", player)
				}
			}
		}
	}
	if object.HasTag("ai") {
		//todo fix performance degradation if intn = 2 ie probability of spawn ~50%
		if rand.Intn(5) <= 1 {
			var bl string
			switch rand.Intn(2) {
			case 0:
				bl = "opel"
			case 1:
				bl = "gun"
			}
			receiver.SpawnManager.SpawnCollectable2(PosAuto, bl, object.(*Unit))
		}

		receiver.mutex.Lock()
		if receiver.spawnedAi == -1 {
			receiver.spawnedAi = receiver.SpawnManager.QuerySpawnedByTagCount("ai")
		} else {
			receiver.spawnedAi--
			if receiver.spawnedAi <= 0 {
				receiver.spawnedAi = receiver.SpawnManager.QuerySpawnedByTagCount("ai")
			}
		}
		receiver.mutex.Unlock()
		if receiver.spawnedAi <= 0 {
			receiver.End(GAME_END_WIN)
		}
	}
}

//async
func (receiver *Game) End(code int) {
	receiver.mutex.Lock()
	if !receiver.inProgress {
		receiver.mutex.Unlock()
		return
	}
	receiver.inProgress = false
	receiver.SpawnManager.DeSpawnAll()

	time.AfterFunc(time.Millisecond*500, func() {
		//todo after despawn callback
		close(receiver.terminator)
		receiver.scenario = nil
		receiver.terminator = nil
		receiver.mutex.Unlock()

		receiver.Trigger(Event{
			EType:   code,
			Object:  nil,
			Payload: nil,
		}, receiver, nil)
	})
}

func NewGame(players []*Player, spm *SpawnManager) (*Game, error) {
	game := &Game{
		players:      players,
		SpawnManager: spm,
		ObservableObject: &ObservableObject{
			Owner:  nil,
			output: make(EventChanel),
		},
		spawnedPlayer: -1,
		spawnedAi:     -1,
	}
	game.ObservableObject.Owner = game
	game.inProgress = false

	return game, nil
}

func (receiver *Game) playerByUnit(unit ObjectInterface) *Player {
	for _, player := range receiver.players {
		if player.Unit == unit {
			return player
		}
	}
	return nil
}

func gameCmdDispatcher(instance *Game, unitEvent EventChanel, terminator <-chan bool) {
	if instance == nil {
		return
	}
	for {
		select {
		case _, ok := <-terminator:
			if !ok {
				return
			}
		case event, ok := <-unitEvent:
			if !ok {
				panic("chanel error")
				return
			}
			if !instance.inProgress {
				continue
			}
			if DEBUG_EVENT {
				logger.Printf("receive Game event %d, %+v", event.EType, event.Object)
			}
			switch event.EType {
			case TANK_EVENT_FIRE:
				go instance.onUnitFire(event.Object.(*Unit), event.Payload)
			case TANK_EVENT_DAMADGE:
				go instance.onUnitDamage(event.Object.(ObjectInterface), event.Payload)
			case OBJECT_EVENT_DESTROY:
				go instance.onObjectDestroy(event.Object.(ObjectInterface), event.Payload)
			case OBJECT_EVENT_DESPAWN:
				go instance.onObjectDeSpawn(event.Object.(ObjectInterface), event.Payload)
			case OBJECT_EVENT_RESET:
				go instance.onObjectReset(event.Object.(ObjectInterface), event.Payload)
			case OBJECT_EVENT_SPAWN:
				go instance.onObjectSpawn(event.Object.(ObjectInterface), event.Payload)
			case COLLECT_EVENT_COLLECTED:
				go instance.onUnitCollect(event.Object.(*Collectable), event.Payload)
			}
		}
	}
}

func scenarioDispatcher(instance *Game, scenarioEvent EventChanel, terminator <-chan bool) {
	if instance == nil {
		return
	}
	for {
		select {
		case _, ok := <-terminator:
			if !ok {
				return
			}
		case event, ok := <-scenarioEvent:
			if !ok {
				return
			}
			if !instance.inProgress {
				continue
			}
			if DEBUG_EVENT {
				logger.Printf("receive scenario event %d, %+v", event.EType, event.Object)
			}
			switch event.EType {
			case SPAWN_REQUEST:
				go instance.onSpawnRequest(event.Object.(*Scenario), event.Payload.(*SpawnRequest))
			}
		}
	}
}

func delayedEnterState(object Stater, state string, delay time.Duration) {
	time.AfterFunc(delay, func() {
		object.Enter(state)
	})
}

type fanoutConfig struct {
	Owner      ObjectInterface
	Direction  Point
	SpeedScale float64
}

func doFanoutSpawn(instance *Game, object ObjectInterface) {
	var bl string
	x, y := object.GetXY()
	coords := []Point{Point{X: -1, Y: -1}, Point{X: 0, Y: -1}, Point{X: 1, Y: -1},
		Point{X: -1, Y: 0}, Point{X: 1, Y: 0},
		Point{X: -1, Y: 1}, Point{X: 0, Y: 1}, Point{X: 1, Y: 1},
	}
	var sscale float64 = 1
	for _, coord := range coords {
		if coord.X == 0 || coord.Y == 0 {
			sscale = .5
		} else {
			sscale = 1.0
		}
		bl, _ = object.GetTagValue("fanout", "blueprint", "projectile-sharp")
		instance.Spawn(Point{
			X: x,
			Y: y,
		}, bl, FanoutProjectileConfigurator, &fanoutConfig{
			Owner:      object,
			Direction:  coord,
			SpeedScale: sscale,
		})
	}
}
