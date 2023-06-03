package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const GAME_START = 200
const GAME_END_WIN = 201
const GAME_END_LOSE = 202

var (
	GameInProgressError         = errors.New("game in progress")
	NoSpawnPointError           = errors.New("no spawn point")
	NotAvailableSpawnPointError = errors.New("not available spawn point")
	NoAvailableLocationManager  = errors.New("location spawn required but no location manager")
)

type Game struct {
	players []*Player
	*SpawnManager
	*ObservableObject
	*Location
	*EffectManager
	*SoundManager
	spawnPoints                []*SpawnPoint
	scenario                   *Scenario
	spawnedPlayer, spawnedAi   int64
	inProgress                 bool
	mutex, delay, aiCountMutex sync.Mutex
	nextDelayedTaskExec        time.Time
	delayedSpawnRequest        []*SpawnRequest
	delayedSpawnRequestCnt     int64
	respawnTimer               *time.Timer
	ctxGame                    context.Context
	ctxCancel                  context.CancelFunc
}

func (receiver *Game) AddPlayer(player *Player) error {
	if receiver.inProgress {
		return GameInProgressError
	}
	receiver.players = append(receiver.players, player)
	return nil
}

func (receiver *Game) GetPlayers() []*Player {
	return receiver.players
}

// MUST RUN async
func (receiver *Game) Run(scenario *Scenario) error {

	receiver.mutex.Lock()
	if receiver.inProgress || scenario == nil {
		receiver.mutex.Unlock()
		return GameInProgressError
	}
	receiver.inProgress = true
	receiver.mutex.Unlock()

	receiver.scenario = scenario
	receiver.spawnedPlayer = 0
	receiver.ctxGame, receiver.ctxCancel = context.WithCancel(context.TODO())

	go scenarioDispatcher(game, scenario.GetEventChanel(), receiver.ctxGame)
	go gameCmdDispatcher(game, game.SpawnManager.UnitEventChanel, receiver.ctxGame)

	err := scenario.Enter("start")

	if err != nil {
		receiver.ctxCancel()
		receiver.scenario = nil
		receiver.SpawnManager.DeSpawnAll(nil)
		receiver.inProgress = false
		logger.Print(err)
		return err
	}

	for pIndex, player := range receiver.players {

		if (pIndex+1)%2 == 0 && scenario.player2Blueprint != "" {
			player.Blueprint = scenario.player2Blueprint
		} else if scenario.player1Blueprint != "" {
			player.Blueprint = scenario.player1Blueprint
		} else {
			player.Blueprint = "player-tank"
		}

		//todo move to custom configurator to remove get info call
		spawnPosition, err := receiver.NewSpawnPosition(player.Blueprint)
		if err != nil {
			logger.Println(fmt.Errorf("unable alloc new position for player %d: %w", pIndex+1, err))
			spawnPosition = PosAuto
		}
		object, err := receiver.SpawnManager.SpawnPlayerTank(spawnPosition, player.Blueprint, player)
		if err != nil {
			logger.Println("at spawning player error: ", err)
		}
		receiver.spawnedPlayer++
		if unit, ok := object.(*Unit); ok && unit.Gun != nil {
			unit.Gun.Current.Name = getProjectilePlDescription(unit.Gun.Current.Projectile).Name
		}
	}

	//timers block
	everyFunc(time.Second/2, receiver.doDelayedSpawn, receiver.ctxGame)

	receiver.playBackground("main")

	//receiver.EffectManager.ApplyGlobalWeather("", 0.01, 0)

	receiver.Trigger(Event{
		EType:   GAME_START,
		Object:  nil,
		Payload: nil,
	}, receiver, nil)

	return nil
}

func (receiver *Game) onSpawnRequest(scenario *Scenario, payload *SpawnRequest) {
	if payload.Count <= 0 {
		payload.Count = 1
	}
	info, err := Info(payload.Blueprint)
	if err != nil {
		logger.Printf("unable to retrieve info about %s \n", payload.Blueprint)
	}
	payload.Info = info
	for i := 0; i < payload.Count; i++ {
		if info.Attributes.Type != "unit" {
			err := receiver.doSpawn(scenario, payload)
			if err != nil {
				logger.Println(err)
			}
		} else if scenario.limits.AiUnits == 0 || scenario.limits.AiUnits > receiver.spawnedAi {
			err := receiver.doSpawn(scenario, payload)
			if err != nil {
				logger.Println(err)
			}
			if err == nil || scenario.limits.AiUnits == 0 {
				continue
			}
		} else {
			receiver.delayedSpawnRequest = append(receiver.delayedSpawnRequest, payload)
			atomic.AddInt64(&receiver.delayedSpawnRequestCnt, 1)
		}
	}
}

func (receiver *Game) doDelayedSpawn() {
	needSpawn := scenario.limits.AiUnits - receiver.spawnedAi
	if needSpawn <= 0 {
		return
	}
	for idx, req := range receiver.delayedSpawnRequest {
		if req == nil {
			continue
		}
		err := receiver.doSpawn(receiver.scenario, req)
		if err == nil {
			atomic.AddInt64(&receiver.delayedSpawnRequestCnt, -1)
			needSpawn--
			receiver.delayedSpawnRequest[idx] = nil
		} else {
			logger.Print(err)
			break
		}
		if needSpawn <= 0 {
			break
		}
	}
}

func (receiver *Game) doSpawn(scenario *Scenario, payload *SpawnRequest) error {
	//todo move to custom configurator to remove get info call
	var spawnPosition Point = payload.Position
	var err error
	if payload.Location != ZoneAuto && payload.Position == PosAuto {
		if receiver.Location != nil {
			spawnPosition, err = receiver.Location.CoordinateByZone(payload.Location)
			if err != nil {
				return fmt.Errorf("unable to locate position by location: %w", err)
			}
			receiver.Location.CapturePoint(spawnPosition, payload.Info.Layer)
		} else {
			return NoAvailableLocationManager
		}
	}
	if spawnPosition == PosAuto {
		spawnPosition, err = receiver.NewSpawnPosition(payload.Blueprint)
		if err != nil {
			logger.Println(fmt.Errorf("unable alloc new position %s: %w", payload.Blueprint, err))
		}
	}
	object, err := receiver.SpawnManager.Spawn(spawnPosition, payload.Blueprint, DefaultConfigurator, payload)
	if err != nil {
		return err
	} else {
		if object.HasTag("ai") {
			atomic.AddInt64(&receiver.spawnedAi, 1)
		}
		if object.HasTag("spawnPoint") {
			receiver.spawnPoints = append(receiver.spawnPoints, object.(*SpawnPoint))
		}
	}
	return nil
}

func (receiver *Game) onUnitFire(object *Unit, payload interface{}) {
	if object.Gun != nil && object.Gun.GetProjectile() != "" {
		_, err := receiver.SpawnManager.SpawnProjectile(PosAuto, object.Gun.GetProjectile(), payload.(FireParams))
		if err != nil {
			logger.Printf("unable to fire %s due: %s \n", object.Gun.GetProjectile(), err)
		} else {
			err = receiver.playSound("fire")
			if err != nil {
				logger.Println(err)
			}
		}
	} else {
		logger.Printf("unable to fire due projectile or gun not found %s \n", object.Gun.GetProjectile())
	}
}

func (receiver *Game) onUnitDamage(object ObjectInterface, payload interface{}) {
	//todo add Damage to interface
	//todo add isHeal flag
	if object.HasTag("wall") || object.HasTag("ice") {
		wall := object.(*Wall)
		wallHp := int(math.Max(float64(wall.HP), 1))
		if wall.FullHP/wallHp >= 2 {
			wall.Enter("damage")
		} else {
			wall.Enter("normal")
		}
	}
	if object.HasTag("highlights-damage") {
		toState, _ := object.GetTagValue("highlights-damage", "moveToState", "receiveDamage")
		returnToState, _ := object.GetTagValue("highlights-damage", "returnToState", ToDefaultState)
		if toState == "" {
			logger.Printf("highlights-damage: toState is empty")
		} else {
			object.(Stater).Enter(toState)
			delayedEnterState(object.(Stater), returnToState, time.Millisecond*500)
		}
	}
	if object.HasTag("tank") {
		tank := object.(*Unit)
		tankHp := int(math.Max(float64(tank.HP), 1))
		if tank.FullHP/tankHp > 2 {
			//wall.Enter("damage") //todo
		}
	}
	if err := receiver.playSound("damage"); err != nil {
		logger.Println(err)
	}
}

func (receiver *Game) onUnitOnSight(object ObjectInterface, payload interface{}) {
	if pObject, ok := payload.(ObjectInterface); ok {
		if !pObject.HasTag("stealth") {
			receiver.SpawnExplosion(PosAuto, "effect-onsight", pObject)
		}
	}
	if unit, ok := payload.(*Unit); ok {
		if unit.GetAttr().AI {
			if bc, ok := unit.Control.(*BehaviorControl); ok {
				bc.See(object.(*Unit))
			}
		}
	}
}

func (receiver *Game) onUnitOffSight(object ObjectInterface, payload interface{}) {
	if pObject, ok := payload.(ObjectInterface); ok {
		if !pObject.HasTag("stealth") && !pObject.GetAttr().Destroyed {
			receiver.SpawnExplosion(PosAuto, "effect-offsight", pObject)
		}
	}
	if unit, ok := payload.(*Unit); ok {
		if unit.GetAttr().AI {
			if bc, ok := unit.Control.(*BehaviorControl); ok {
				bc.UnSee(object.(*Unit))
			}
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
		toState, _ := object.GetTagValue("highlights-appear", "moveToState", "appear")
		returnToState, _ := object.GetTagValue("highlights-appear", "returnToState", ToDefaultState)
		durStr, _ := object.GetTagValue("highlights-appear", "duration", "8000000000")
		duration, err := strconv.Atoi(durStr)
		if err != nil || toState == "" {
			logger.Printf("highlights-appear: invalid duration value %s or toState value %s", durStr, toState)
		} else {
			object.(Stater).Enter(toState)
			delayedEnterState(object.(Stater), returnToState, time.Duration(duration))
		}
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

	if payload != nil && payload.(ObjectInterface).HasTag("water") {
		//EnvironmentDamage by water special case //todo make some better
		_, err := receiver.SpawnManager.SpawnExplosion(PosAuto, "tank-base-sunk", object)
		if err != nil {
			logger.Printf("unable to spawn explosion[sunk]: %s \n", err)
		}
		if err := receiver.playSound("damage"); err != nil {
			logger.Println(err)
		}
	} else if object.HasTag("explosive") {
		bl, _ := object.GetTagValue("explosive", "blueprint", "tank-base-explosion")
		_, err := receiver.SpawnManager.SpawnExplosion(PosAuto, bl, object)
		if err != nil {
			logger.Printf("unable to spawn explosion: %s \n", err)
		} else {
			if err = receiver.playSound("explosion"); err != nil {
				logger.Println(err)
			}
			if err = receiver.EffectManager.ApplyGlobalShake(0.3, time.Second*1); err != nil {
				logger.Println(err)
			}
		}
		if err = receiver.playSound("explosion"); err != nil {
			logger.Println(err)
		}
	} else {
		if object.HasTag("obstacle") {
			if err := receiver.playSound("damage"); err != nil {
				logger.Println(err)
			}
		}
	}

	if object.HasTag("ice") {
		bl, _ := object.GetTagValue("ice", "blueprint", "ice-water")
		point := object.GetXY()
		originalBl := object.GetAttr().Blueprint
		originalTeam := object.(ObjectInterface).GetAttr().Team
		newWaterObject, err := receiver.SpawnManager.Spawn(point, bl, DefaultConfigurator, &SpawnRequest{
			Team: originalTeam,
		})
		if err != nil {
			logger.Printf("unable to spawn water: %s \n", err)
		}
		time.AfterFunc(time.Second*time.Duration(30), func() { //time.AfterFunc(time.Second*time.Duration(rand.Intn(11)+25), func() {
			if game.inProgress && !newWaterObject.GetAttr().Destroyed {
				_, err := receiver.SpawnManager.Spawn(point, originalBl, func(object ObjectInterface, config interface{}) ObjectInterface {
					DefaultConfigurator(object, config)
					//probably not best way to do this, mb wait until spawned
					object.(Vulnerable).ReciveDamage(&DamageProxy{Damage: object.(*Wall).FullHP - 1, Tags: &Tags{}, From: object})
					return object
				}, &SpawnRequest{
					Team: originalTeam,
				})
				if err != nil {
					logger.Printf("unable to respawn ice: %s \n", err)
				} else {
					receiver.SpawnManager.DeSpawn(newWaterObject)
				}
			}
		})
	}

	if object.HasTag("fanout") {
		if err := doFanoutSpawn(receiver, object); err != nil {
			logger.Println(err)
		}
	}

	if object.HasTag("highlights-disappear") {
		despawnNow = false
		toState, _ := object.GetTagValue("highlights-disappear", "moveToState", "disappear")
		durStr, _ := object.GetTagValue("highlights-disappear", "duration", "1000000000")
		duration, err := strconv.Atoi(durStr)
		if err != nil || toState == "" {
			logger.Printf("highlights-appear: invalid duration value %s or toState value %s", durStr, toState)
		} else {
			object.(Stater).Enter(toState)
			time.AfterFunc(time.Duration(duration), func() {
				if receiver.inProgress { //todo fix in game method
					receiver.SpawnManager.DeSpawn(object)
				}
			})
		}
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
			bl := GetConventionalProjectileName()
			unit.Gun.Upgrade(&GunState{
				Projectile:       bl,
				Name:             getProjectilePlDescription(bl).Name,
				Ammo:             10,
				ShotQueue:        1,
				PerShotQueueTime: time.Second / 5,
				ReloadTime:       2 * time.Second,
				lastShotTime:     time.Time{},
			})

		}
		if seed >= 9 {
			unit.Gun.Upgrade(&GunState{
				Projectile:       "tank-base-projectile-apocalypse",
				Name:             getProjectilePlDescription("tank-base-projectile-apocalypse").Name,
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

func (receiver *Game) onSpawnPointStatus(object *SpawnPoint, payload interface{}) {
	//nope for now
}

func (receiver *Game) onObjectDeSpawn(object ObjectInterface, payload interface{}) {
	if object.HasTag("base") {
		receiver.End(GAME_END_LOSE)
	}
	if object.HasTag("player") {
		for idx, player := range receiver.players {
			if player.Unit == object {
				left := player.DecrRetry(1)
				logger.Printf("cycleId: %d, player %d have %d retry\n", CycleID, idx+1, left)
				if left <= 0 {
					if atomic.AddInt64(&receiver.spawnedPlayer, -1) == 0 {
						time.AfterFunc(time.Second, func() { //small delay to improve experience
							receiver.End(GAME_END_LOSE)
						})
					}
				} else {
					if receiver.inProgress {
						//todo theoretical may cause game freeze due send signal on closed dispatcher
						spawnPosition, err := receiver.NewSpawnPosition(player.Blueprint)
						if err != nil {
							logger.Println(fmt.Errorf("player %d: %w", idx+1, err))
							spawnPosition = PosAuto //no position?
						}
						if _, err = receiver.SpawnManager.SpawnPlayerTank(spawnPosition, player.Blueprint, player); err != nil {
							logger.Println("at respawning player error: ", err)
						}
					}
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
			receiver.SpawnManager.SpawnCollectable(PosAuto, bl, object.(*Unit))
		}

		if v := receiver.DecrSpawnedAi(); v == 0 && receiver.delayedSpawnRequestCnt <= 0 {
			receiver.End(GAME_END_WIN)
		}
	}
}

func (receiver *Game) DecrSpawnedAi() int64 {
	return atomic.AddInt64(&receiver.spawnedAi, -1)
}

// async
func (receiver *Game) End(code int) {
	if !receiver.inProgress {
		return
	}
	receiver.mutex.Lock()
	if !receiver.inProgress {
		return
	}
	if DEBUG_SHUTDOWN {
		logger.Println("starting of the END")
	}
	receiver.inProgress = false
	receiver.EffectManager.CancelAllEffects()
	receiver.stopBackground("main")
	if DEBUG_SHUTDOWN {
		logger.Println("begining despawn ALL")
	}

	receiver.SpawnManager.DeSpawnAll(func() {
		//todo after despawn callback
		if DEBUG_SHUTDOWN {
			logger.Println("despawn ALL complete")
		}
		receiver.ctxCancel()

		if DEBUG_SHUTDOWN {
			logger.Println("dispatcher shutdown")
		}

		receiver.scenario = nil
		receiver.mutex.Unlock()

		if DEBUG_SHUTDOWN {
			logger.Println("END complete, trigger event")
		}

		receiver.Trigger(Event{
			EType:   code,
			Object:  nil,
			Payload: nil,
		}, receiver, nil)
	})
}

func (receiver *Game) NewSpawnPosition(blueprint string) (Point, error) {
	info, err := Info(blueprint)
	if err != nil {
		return NoPos, err
	}
	if point, err := receiver.CaptureRandomSpawnPoint(blueprint); err == nil {
		if receiver.location != nil {
			receiver.Location.CapturePoint(point.GetXY(), info.Layer)
		}
		return point.GetXY(), nil
	} else {
		if !errors.Is(err, NoSpawnPointError) {
			//return NoPos, err //todo check if allowed spawn point exist
		}
		if receiver.Location != nil {
			return receiver.Location.Coordinate2Spawn(true, info.Layer)
		} else {
			//no need to capture this :(
			return Point{}, nil
		}
	}
}

func (receiver *Game) CaptureRandomSpawnPoint(blueprint string) (*SpawnPoint, error) {
	if len(receiver.spawnPoints) == 0 {
		return nil, NoSpawnPointError
	}
	info, err := Info(blueprint)
	if err != nil {
		return nil, err
	}
	available := make([]*SpawnPoint, 0, len(receiver.spawnPoints))
	for _, point := range receiver.spawnPoints {
		if point.IsAvailable() && point.CanSpawn(info.Tags) {
			available = append(available, point)
		}
	}
	avlLen := len(available)
	if avlLen == 0 {
		return nil, NotAvailableSpawnPointError
	}
	start := rand.Intn(avlLen)
	for i := start; i < avlLen; i++ {
		if available[i].Capture() {
			logger.Println("point captured", available[i].GetXY(), available[i].ID)
			return available[i], nil
		} else {
			logger.Println("point already captured")
		}
	}
	for i := start; i >= 0; i-- {
		if available[i].Capture() {
			logger.Println("point captured", available[i].GetXY(), available[i].ID)
			return available[i], nil
		} else {
			logger.Println("point already captured")
		}
	}
	return nil, NotAvailableSpawnPointError
}

func (receiver *Game) playSound(key string) error {
	if receiver.SoundManager != nil {
		return receiver.SoundManager.Play(key)
	}
	return nil
}

func (receiver *Game) playBackground(key string) {
	if receiver.SoundManager != nil {
		receiver.SoundManager.Play(key)
	}
}

func (receiver *Game) stopBackground(key string) {
	logger.Println("stopping of bg playing not implemented")
}

func NewGame(players []*Player, spm *SpawnManager) (*Game, error) {
	game = &Game{
		players:      players,
		SpawnManager: spm,
		ObservableObject: &ObservableObject{
			Owner:  nil,
			output: make(EventChanel),
		},
		spawnedPlayer: 0,
		spawnedAi:     0,
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

func gameCmdDispatcher(instance *Game, unitEvent EventChanel, ctx context.Context) {
	if instance == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
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
			case UNIT_EVENT_FIRE:
				go instance.onUnitFire(event.Object.(*Unit), event.Payload)
			case UNIT_EVENT_DAMAGE:
				go instance.onUnitDamage(event.Object.(ObjectInterface), event.Payload)
			case UNIT_EVENT_ONSIGTH:
				go instance.onUnitOnSight(event.Object.(ObjectInterface), event.Payload)
			case UNIT_EVENT_OFFSIGTH:
				go instance.onUnitOffSight(event.Object.(ObjectInterface), event.Payload)
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
			case SPAWN_POINT_STATUS:
				go instance.onSpawnPointStatus(event.Object.(*SpawnPoint), event.Payload)
			}
		}
	}
}
func scenarioDispatcher(instance *Game, scenarioEvent EventChanel, ctx context.Context) {
	if instance == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
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
				//sync due t
				instance.onSpawnRequest(event.Object.(*Scenario), event.Payload.(*SpawnRequest))
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

func doFanoutSpawn(instance *Game, object ObjectInterface) error {
	var bl string
	pos := object.GetXY()
	coords := []Point{Point{X: -1, Y: -1}, Point{X: 0, Y: -1}, Point{X: 1, Y: -1},
		Point{X: -1, Y: 0}, Point{X: 1, Y: 0},
		Point{X: -1, Y: 1}, Point{X: 0, Y: 1}, Point{X: 1, Y: 1},
	}
	var sscale float64 = 1
	var err error
	for _, coord := range coords {
		if coord.X == 0 || coord.Y == 0 {
			sscale = .5
		} else {
			sscale = 1.0
		}
		bl, _ = object.GetTagValue("fanout", "blueprint", "projectile-sharp")
		if err != nil {
			break
		}
		_, err = instance.Spawn(pos, bl, FanoutProjectileConfigurator, &fanoutConfig{
			Owner:      object,
			Direction:  coord,
			SpeedScale: sscale,
		})
	}
	return err
}
