package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	"strconv"
	"time"
)

type Package struct {
	M             map[string]Loader
	FilePath      string
	FileExtension string
}

func NewJsonPackage() *Package {
	instance := new(Package)
	instance.M = make(map[string]Loader)

	instance.FilePath = "blueprint"
	instance.FileExtension = "json"

	instance.M["/"] = RootLoader
	instance.M["unit"] = UnitLoader
	instance.M["wall"] = WallLoader
	instance.M["explosion"] = ExplosionLoader
	instance.M["projectile"] = ProjectileLoader
	instance.M["collectable"] = CollectableLoader
	instance.M["spawnPoint"] = SpawnPointLoader
	instance.M["gun"] = GunLoader
	instance.M["motionObject"] = MotionObjectLoader
	instance.M["controlledObject"] = ControlledObjectLoader
	instance.M["object"] = ObjectLoader
	instance.M["tags"] = TagsLoader
	instance.M["state"] = StateLoader
	instance.M["stateItem"] = StateItemLoader
	instance.M["collision"] = CollisionLoader
	instance.M["spriter"] = SpriterLoader
	instance.M["sprite"] = SpriteLoader
	instance.M["animation"] = AnimationLoader
	instance.M["composition"] = CompositionLoader

	return instance
}

func lGetObject(ctx context.Context, blueprint string, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) (interface{}, error) {
	loader := get(blueprint)
	if loader == nil {
		return nil, fmt.Errorf("%s: %w", blueprint, LoaderNotFoundError)
	}
	object := loader(ctx, get, collector, preset, payload)
	if object == nil {
		return nil, fmt.Errorf("%s: %w", blueprint, InstanceError)
	}
	return object, nil
}

func RootLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	requireBytes, dType, _, _ := jsonparser.Get(payload, "require")
	requireList := make([]string, 0)
	if dType == jsonparser.String || dType == jsonparser.Array {
		var requireFunc RequireFunc
		if obj, err := lGetObject(ctx, "require", get, collector, preset, payload); !collector.Add(err) {
			requireFunc = obj.(RequireFunc)
			if dType == jsonparser.String {
				err := requireFunc(string(requireBytes))
				if err != nil {
					collector.Add(fmt.Errorf("unable to require %s: %w", requireBytes, err))
					return nil
				} else {
					requireList = append(requireList, string(requireBytes))
				}
			} else {
				var requireError error
				jsonparser.ArrayEach(requireBytes, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					if requireError != nil {
						return
					}
					if dataType != jsonparser.String {
						collector.Add(fmt.Errorf("require skip value, it has ubnormal format %w", ParseError))
						return
					}
					requireError = requireFunc(string(value))
					if requireError != nil {
						collector.Add(fmt.Errorf("unable to require %s: %w", string(value), requireError))
						return
					} else {
						requireList = append(requireList, string(value))
					}
				})
				if requireError != nil {
					return nil
				}
			}
		}
	} else if dType == jsonparser.Null || dType == jsonparser.NotExist {
		//none
	} else {
		collector.Add(fmt.Errorf("require has ubnormal format %w", ParseError))
	}

	uType, err := jsonparser.GetString(payload, "type")
	if collector.Add(err) {
		return nil
	}
	object, err := lGetObject(ctx, uType, get, collector, preset, payload)
	if collector.Add(err) {
		return nil
	}
	if oi, ok := object.(ObjectInterface); ok {
		oi.GetAttr().Type = uType
		oi.GetAttr().Require = requireList
	}

	return object
}

func UnitLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		output     EventChanel
		object     *MotionObject
		stateObj   *State
		oo         *ObservableObject
		co         *ControlledObject
		unit       *Unit
		gun        *Gun
		vision     *collider.ClBody
		behaviorAi bool
		err        error
	)

	if obj, err := lGetObject(ctx, "motionObject", get, collector, preset, payload); !collector.Add(err) {
		object = obj.(*MotionObject)
	} else {
		return nil
	}

	//skip error because of dataType validation
	stateCfg, dType, _, _ := jsonparser.Get(payload, "state")
	switch dType {
	case jsonparser.String:
		stateBytes, err := loadState(string(stateCfg))
		if !collector.Add(err) {
			if obj, err := lGetObject(ctx, "state", get, collector, SpriteerConfig{
				Custom: object.Attributes.Custom,
			}, stateBytes); !collector.Add(err) {
				stateObj = obj.(*State)
			}
		}
	case jsonparser.Object:
		if obj, err := lGetObject(ctx, "state", get, collector, SpriteerConfig{
			Custom: object.Attributes.Custom,
		}, stateCfg); !collector.Add(err) {
			stateObj = obj.(*State)
		}
	case jsonparser.Null:
		fallthrough
	case jsonparser.NotExist:
		//none
	default:
		collector.Add(fmt.Errorf("state: %w", ParseError))
	}

	//skip error because of dataType validation
	gunBytes, dType, _, _ := jsonparser.Get(payload, "gun")
	switch dType {
	case jsonparser.Object:
		if object, err := lGetObject(ctx, "gun", get, collector, preset, gunBytes); !collector.Add(err) {
			gun = object.(*Gun)
		}
	case jsonparser.Null:
		fallthrough
	case jsonparser.NotExist:
		//none
	default:
		collector.Add(fmt.Errorf("gun: %w", ParseError))
	}

	if !DEBUG_DISABLE_VISION {
		visionBytes, dType, _, _ := jsonparser.Get(payload, "vision")
		switch dType {
		case jsonparser.Object:
			size := Size{}
			json.Unmarshal(visionBytes, &size)
			vision = collider.NewPenetrateCollision(0, 0, size.W, size.H)
		case jsonparser.Null:
		case jsonparser.NotExist:
		default:
			collector.Add(fmt.Errorf("vision: %w", ParseError))
		}
	}

	//todo remove
	if obj, err := lGetObject(ctx, "eventChanel", get, collector, preset, payload); !collector.Add(err) {
		output = obj.(EventChanel)
	}

	oo, err = NewObservableObject(output, nil)
	if !collector.Add(err) {
		if !DEBUG_NO_AI {
			//skip error because of dataType validation
			_, dataType, _, _ := jsonparser.Get(payload, "control")
			switch dataType {
			case jsonparser.Null:
				fallthrough
			case jsonparser.NotExist:
				obj, _ := controller.NewAIControl()
				co, err = NewControlledObject(obj, nil)
			default:
				if obj, err := lGetObject(ctx, "controlledObject", get, collector, preset, payload); !collector.Add(err) {
					co = obj.(*ControlledObject)
					if _, ok := co.Control.(*BehaviorControl); ok {
						behaviorAi = true
					}
				}
			}
		} else {
			obj, _ := controller.NewNoneControl()
			co, err = NewControlledObject(obj, nil)
		}
		if !collector.Add(err) {
			unit, err = NewUnit(co, oo, object, stateObj, vision)
			unit.ObservableObject.Owner = unit
		}
		if !collector.Add(err) {

		}
	}

	if unit != nil {

		unit.Gun = gun
		unit.Attributes.Obstacle = true
		unit.Attributes.Vulnerable = true
		unit.Attributes.Motioner = true
		unit.Attributes.Evented = true
		unit.Attributes.Controled = true
		if behaviorAi {
			unit.Attributes.AI = true
		}
		if unit.vision != nil {
			//todo remove
			unit.Attributes.Visioned = true
		}

		if object.HasTag("vulnerable") {
			hp, err := jsonparser.GetInt(payload, "hp")
			if err == nil {
				unit.FullHP = int(hp)
				unit.HP = int(hp)
			} else {
				collector.Add(fmt.Errorf("no hp value set %w", err))
			}
		}

		if unit.HasTag("scored") {
			score, err := jsonparser.GetInt(payload, "score")
			if err == nil {
				unit.Score = int(score)
			} else {
				collector.Add(fmt.Errorf("no score value set %w", err))
			}
		}
	}

	return unit
}

func WallLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		object   *Object
		stateObj *State
		oo       *ObservableObject
		wall     *Wall
		output   EventChanel
		err      error
	)

	if obj, err := lGetObject(ctx, "object", get, collector, preset, payload); !collector.Add(err) {
		object = obj.(*Object)
	} else {
		return nil
	}

	//skip error because of dataType validation
	stateCfg, dType, _, _ := jsonparser.Get(payload, "state")
	switch dType {
	case jsonparser.String:
		//todo rename
		stateBytes, err := loadState(string(stateCfg))
		if !collector.Add(err) {
			if obj, err := lGetObject(ctx, "state", get, collector, SpriteerConfig{
				Custom: object.Attributes.Custom,
			}, stateBytes); !collector.Add(err) {
				stateObj = obj.(*State)
			}
		}
	case jsonparser.Object:
		if obj, err := lGetObject(ctx, "state", get, collector, SpriteerConfig{
			Custom: object.Attributes.Custom,
		}, stateCfg); !collector.Add(err) {
			stateObj = obj.(*State)
		}
	case jsonparser.Null:
		fallthrough
	case jsonparser.NotExist:
		//none
	default:
		collector.Add(fmt.Errorf("state: %w", ParseError))
	}

	//todo remove
	if obj, err := lGetObject(ctx, "eventChanel", get, collector, preset, payload); !collector.Add(err) {
		output = obj.(EventChanel)
	}

	oo, err = NewObservableObject(output, nil)

	if !collector.Add(err) {
		wall, err = NewWall(object, stateObj, oo)
		wall.ObservableObject.Owner = wall
		collector.Add(err)
	}

	if wall != nil {
		wall.Attributes.Obstacle = true
		wall.Attributes.Vulnerable = true
		wall.Attributes.Evented = true

		if wall.HasTag("vulnerable") {
			hp, err := jsonparser.GetInt(payload, "hp")
			if err == nil {
				wall.FullHP = int(hp)
				wall.HP = int(hp)
			} else {
				collector.Add(fmt.Errorf("no hp value set %w", err))
			}
		}

		if wall.HasTag("scored") {
			score, err := jsonparser.GetInt(payload, "score")
			if err == nil {
				wall.Score = int(score)
			} else {
				collector.Add(fmt.Errorf("no score value set %w", err))
			}
		}
	}

	if regeneration, err := jsonparser.GetFloat(payload, "regeneration"); !collector.Add(err) {
		if regeneration > 0.0 {
			wall.Regenerator, _ = NewRegenerator(regeneration)
		}
	}

	return wall
}

func CollectableLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		object   *Object
		stateObj *State
		oo       *ObservableObject
		collect  *Collectable
		output   EventChanel
		err      error
	)

	if obj, err := lGetObject(ctx, "object", get, collector, preset, payload); !collector.Add(err) {
		object = obj.(*Object)
	} else {
		return nil
	}

	//skip error because of dataType validation
	stateCfg, dType, _, _ := jsonparser.Get(payload, "state")
	switch dType {
	case jsonparser.String:
		//todo rename
		stateBytes, err := loadState(string(stateCfg))
		if !collector.Add(err) {
			if obj, err := lGetObject(ctx, "state", get, collector, SpriteerConfig{
				Custom: object.Attributes.Custom,
			}, stateBytes); !collector.Add(err) {
				stateObj = obj.(*State)
			}
		}
	case jsonparser.Object:
		if obj, err := lGetObject(ctx, "state", get, collector, SpriteerConfig{
			Custom: object.Attributes.Custom,
		}, stateCfg); !collector.Add(err) {
			stateObj = obj.(*State)
		}
	case jsonparser.Null:
		fallthrough
	case jsonparser.NotExist:
		//none
	default:
		collector.Add(fmt.Errorf("state: %w", ParseError))
	}

	//todo remove
	if obj, err := lGetObject(ctx, "eventChanel", get, collector, preset, payload); !collector.Add(err) {
		output = obj.(EventChanel)
	}

	oo, err = NewObservableObject(output, nil)

	if !collector.Add(err) {
		collect, err = NewCollectable(object, oo, stateObj, nil)
		collect.ObservableObject.Owner = collect
		collector.Add(err)
	}

	if collect != nil {
		collect.Attributes.Obstacle = true
		collect.Attributes.Evented = true

		if collect.HasTag("ttl") {
			ttl, err := jsonparser.GetInt(payload, "ttl")
			if err == nil {
				collect.Ttl = time.Duration(ttl)
			} else {
				collector.Add(fmt.Errorf("no ttl %w", err))
			}
		}
	}

	return collect
}

func ExplosionLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		object    *Object
		oo        *ObservableObject
		explosion *Explosion
		output    EventChanel
		err       error
	)

	if obj, err := lGetObject(ctx, "object", get, collector, preset, payload); !collector.Add(err) {
		object = obj.(*Object)
	} else {
		return nil
	}

	//todo remove
	if obj, err := lGetObject(ctx, "eventChanel", get, collector, preset, payload); !collector.Add(err) {
		output = obj.(EventChanel)
	}

	oo, err = NewObservableObject(output, nil)

	if !collector.Add(err) {
		explosion, err = NewExplosion2(object, oo, nil)
		explosion.ObservableObject.Owner = explosion
		collector.Add(err)
	}

	if explosion != nil {
		explosion.Attributes.Evented = true
		explosion.Attributes.Danger = true

		if explosion.HasTag("ttl") {
			ttl, err := jsonparser.GetInt(payload, "ttl")
			if err == nil {
				explosion.Ttl = time.Duration(ttl)
			} else {
				collector.Add(fmt.Errorf("no ttl %w", err))
			}
		}

		damage, err := jsonparser.GetInt(payload, "damage")
		if !collector.Add(err) {
			explosion.Damage = int(damage)
		}
		damage, err = jsonparser.GetInt(payload, "dotDamage")
		if err == nil {
			explosion.DotDamage = int(damage)
		}
	}

	return explosion
}

func ProjectileLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		object     *MotionObject
		stateObj   *State
		oo         *ObservableObject
		projectile *Projectile
		output     EventChanel
		err        error
	)

	if obj, err := lGetObject(ctx, "motionObject", get, collector, preset, payload); !collector.Add(err) {
		object = obj.(*MotionObject)
	} else {
		return nil
	}

	//skip error because of dataType validation
	stateCfg, dType, _, _ := jsonparser.Get(payload, "state")
	switch dType {
	case jsonparser.String:
		stateBytes, err := loadState(string(stateCfg))
		if !collector.Add(err) {
			if obj, err := lGetObject(ctx, "state", get, collector, SpriteerConfig{
				Custom: object.Attributes.Custom,
			}, stateBytes); !collector.Add(err) {
				stateObj = obj.(*State)
			}
		}
	case jsonparser.Object:
		if obj, err := lGetObject(ctx, "state", get, collector, SpriteerConfig{
			Custom: object.Attributes.Custom,
		}, stateCfg); !collector.Add(err) {
			stateObj = obj.(*State)
		}
	case jsonparser.Null:
		fallthrough
	case jsonparser.NotExist:
		//none
	default:
		collector.Add(fmt.Errorf("state: %w", ParseError))
	}

	//todo remove
	if obj, err := lGetObject(ctx, "eventChanel", get, collector, preset, payload); !collector.Add(err) {
		output = obj.(EventChanel)
	}

	oo, err = NewObservableObject(output, nil)

	if !collector.Add(err) {
		projectile, err = NewProjectile2(object, oo, stateObj, nil)
		projectile.ObservableObject.Owner = projectile
		if !collector.Add(err) {

		}
	}

	if projectile != nil {
		projectile.Attributes.Obstacle = true
		projectile.Attributes.Vulnerable = true
		projectile.Attributes.Motioner = true
		projectile.Attributes.Evented = true
		projectile.Attributes.Controled = true

		if projectile.HasTag("ttl") {
			ttl, err := jsonparser.GetInt(payload, "ttl")
			if err == nil {
				projectile.Ttl = time.Duration(ttl)
			} else {
				collector.Add(fmt.Errorf("no ttl %w", err))
			}
		}

		damage, err := jsonparser.GetInt(payload, "damage")
		if !collector.Add(err) {
			projectile.Damage = int(damage)
		}

		damage, err = jsonparser.GetInt(payload, "dotDamage")
		if err == nil {
			projectile.DotDamage = int(damage)
		}
	}

	return projectile
}

func SpawnPointLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		object *Object
		output EventChanel
	)

	if obj, err := lGetObject(ctx, "object", get, collector, preset, payload); !collector.Add(err) {
		object = obj.(*Object)
	} else {
		return nil
	}

	//todo remove
	if obj, err := lGetObject(ctx, "eventChanel", get, collector, preset, payload); !collector.Add(err) {
		output = obj.(EventChanel)
	}

	oo, _ := NewObservableObject(output, nil)
	sp, _ := NewSpawnPoint(object, oo)

	if allowListBytes, dType, _, _ := jsonparser.Get(payload, "allow"); dType == jsonparser.Array {
		index := 0
		jsonparser.ArrayEach(allowListBytes, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if dataType == jsonparser.String {
				sp.AddAllowing(string(value))
			} else if dataType == jsonparser.Array {
				index2 := 0
				allowing := make([]string, 0, 0)
				jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					if dataType != jsonparser.String {
						collector.Add(fmt.Errorf("allow value %d:%d unknown format: %w", index, index2, ParseError))
						return
					}
					allowing = append(allowing, string(value))
				})
				sp.AddAllowing(allowing...)
			} else {
				collector.Add(fmt.Errorf("allow value %d unknown format: %w", index, ParseError))
			}
			index++
		})
	} else if dType != jsonparser.Null || dType != jsonparser.NotExist {
		collector.Add(fmt.Errorf("allow list unknown format: %w", ParseError))
	}

	if sp != nil {
		sp.Attributes.Evented = true
	}

	return sp
}

func GunLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		object *Gun
		err    error
	)

	object, err = NewGun(nil)
	collector.Add(err)
	gunState := new(GunState)
	gunState.Ammo = -1
	gunState.ShotQueue = 1
	gunState.ReloadTime = 1
	gunState.PerShotQueueTime = time.Second / 5 //to nany? to short?

	if !collector.Add(json.Unmarshal(payload, gunState)) {
		//onlyBasic for now
		object.Basic(gunState)
	}

	return object
}

func MotionObjectLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		object *MotionObject
		config *MotionObjectConfig
	)

	if obj, err := lGetObject(ctx, "object", get, collector, preset, payload); !collector.Add(err) {
		if cfg, ok := preset.(*MotionObjectConfig); ok { //todo separate defaults and params
			config = cfg
		} else {
			config = new(MotionObjectConfig)
		}
		collector.Add(json.Unmarshal(payload, config))
		if config.Direction.X == 0 && config.Direction.Y == 0 {
			config.Direction.Y = -1
		}
		_, spdMin, _, _ := jsonparser.Get(payload, "speed", "min")
		_, spdMax, _, _ := jsonparser.Get(payload, "speed", "max")
		object, err = NewMotionObject(obj.(*Object), config.Direction, Point{
			X: config.Speed.Min,
			Y: config.Speed.Min,
		})
		collector.Add(err)
		if spdMin == jsonparser.Number {
			object.MinSpeed.X = config.Speed.Min
			object.MinSpeed.Y = config.Speed.Min
		}
		if spdMax == jsonparser.Number {
			object.MaxSpeed.X = config.Speed.Max
			object.MaxSpeed.Y = config.Speed.Max
		}
		if config.AccelTimeFunc != "" {
			tf, err := GetTimeFunc(config.AccelTimeFunc)
			if !collector.Add(err) {
				object.AccelTimeFunc = tf
			}
		}
		if config.AccelTime > 0 {
			object.AccelDuration = config.AccelTime
		}

	}

	if object != nil {
		object.Attributes.Motioner = true
	}

	return object
}

func ControlledObjectLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		object *ControlledObject
	)

	if obj, err := lGetObject(ctx, "ai", get, collector, preset, payload); !collector.Add(err) {
		object, _ = NewControlledObject(obj.(controller.Controller), nil)
	} else {
		collector.Add(fmt.Errorf("%s: %w", "ai", LoaderNotFoundError))
		simpl, _ := controller.NewAIControl()
		object, _ = NewControlledObject(simpl, nil)
	}

	if object == nil {
		obj, _ := controller.NewNoneControl()
		object, _ = NewControlledObject(obj, nil)
	}

	return object
}

func ObjectLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		sprite    Spriteer
		collision *collider.ClBody
		cfg       *GameConfig
		custom    CustomizeMap
		tags      *Tags
		err       error
	)

	_, dType, _, _ := jsonparser.Get(payload, "animation")
	if dType != jsonparser.NotExist {
		collector.Add(fmt.Errorf("animation key is depricated for jsonLoaders: %w", ParseError))
	}

	if obj, err := lGetObject(ctx, "gameConfig", get, collector, preset, payload); !collector.Add(err) {
		cfg = obj.(*GameConfig)
	}

	tagsBytes, dType, _, _ := jsonparser.Get(payload, "tags")
	if dType != jsonparser.Null || dType != jsonparser.NotExist {
		if obj, err := lGetObject(ctx, "tags", get, collector, preset, tagsBytes); !collector.Add(err) {
			tags = obj.(*Tags)
		}
	}

	if !cfg.disableCustomization {
		customBytes, dataType, _, _ := jsonparser.Get(payload, "custom")
		switch dataType {
		case jsonparser.Object:
			custom = make(CustomizeMap)
			collector.Add(json.Unmarshal(customBytes, &custom))
		case jsonparser.Null:
			fallthrough
		case jsonparser.NotExist:
			//none
		default:
			collector.Add(fmt.Errorf("custom: %w", ParseError))
		}
	}

	//skip error because of dataType validation
	spriteCfg, dType, _, _ := jsonparser.Get(payload, "sprite")
	switch dType {
	case jsonparser.String:
		if sprite, err = GetSprite2(string(spriteCfg)); err != nil {
			if sprite, err = LoadSprite2(string(spriteCfg), false); !collector.Add(err) {
				collector.Add(AddSprite(string(spriteCfg), sprite.(*Sprite)))
			}
		}
	case jsonparser.Object:
		if obj, err := lGetObject(ctx, "spriter", get, collector, SpriteerConfig{
			Custom: custom,
		}, spriteCfg); !collector.Add(err) {
			sprite = obj.(Spriteer)
		}
	default:
		collector.Add(fmt.Errorf("sprite: %w", ParseError))
	}

	//skip error because of dataType validation
	collisionCfg, dType, _, _ := jsonparser.Get(payload, "collision")
	switch dType {
	case jsonparser.Object:
		if obj, err := lGetObject(ctx, "collision", get, collector, preset, collisionCfg); !collector.Add(err) {
			collision = obj.(*collider.ClBody)
		}
	case jsonparser.Null:
		fallthrough
	case jsonparser.NotExist:
		//nope if size and proper tags are set
	default:
		collector.Add(fmt.Errorf("collision: %w", ParseError))
	}

	object, err := NewObject(sprite, collision)

	if object != nil {

		object.Attributes.ID = genId()
		object.Attributes.Tagable = true
		if collision != nil {
			object.Attributes.Collided = true
		}
		if sprite != nil {
			object.Attributes.Renderable = true
		}

		object.Tags = tags

		//size
		if sizePl, dt, _, _ := jsonparser.Get(payload, "size"); dt == jsonparser.Object {
			if collision != nil {
				collector.Add(fmt.Errorf("only one of collision|size must be set: %w", ParseError))
			} else {
				size := Size{}
				collector.Add(json.Unmarshal(sizePl, &size))
				if object.HasTag("nocolision") {
					object.collision = collider.NewFakeCollision(0, 0, size.W, size.H)
				} else if object.HasTag("projectile") || object.HasTag("explosion") || object.HasTag("penetrate") {
					object.collision = collider.NewPenetrateCollision(0, 0, size.W, size.H)
				} else if object.HasTag("obstacle") {
					object.collision = collider.NewCollision(0, 0, size.W, size.H)
				} else if !object.HasTag("obstacle") && object.HasTag("vulnerable") {
					object.collision = collider.NewPenetrateCollision(0, 0, size.W, size.H)
				} else {
					logger.Printf("object %d %s has default fake collision\n", object.ID, payload)
					object.collision = collider.NewFakeCollision(0, 0, size.W, size.H)
				}
				if object.HasTag("static") {
					object.collision.SetStatic(true)
				}
			}
		} else if collision == nil {
			collector.Add(fmt.Errorf("size or collision must be set: %w", ParseError))
		}

		zIndex, err := jsonparser.GetInt(payload, "zIndex")
		if err == nil {
			object.zIndex = int(zIndex)
		}

		name, err := jsonparser.GetString(payload, "name")
		if err == nil {
			object.GetAttr().Name = name
		}

		descr, err := jsonparser.GetString(payload, "description")
		if err == nil {
			object.GetAttr().Description = descr
		}

		if object.HasTag("tracked") {
			object.Tracker, err = NewTracker()
			collector.Add(err)
		}

		if object.HasTag("terrain") {
			object.Attributes.Layer = LOCATION_LAYER_TERRAIN
		} else if object.HasTag("air") {
			object.Attributes.Layer = LOCATION_LAYER_AIR
		}

		if custom != nil {
			object.Attributes.Custom = custom
		}
	}

	return object
}

func TagsLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	tags, _ := NewTags()
	//skip error because of dataType validation
	tagsCfg, dType, _, _ := jsonparser.Get(payload)
	switch dType {
	case jsonparser.Array:
		jsonparser.ArrayEach(tagsCfg, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			switch dataType {
			case jsonparser.String:
				tags.addTag(string(value))
			case jsonparser.Object:
				strVal, err := jsonparser.GetString(value, "name")
				if err != nil || strVal == "" {
					collector.Add(fmt.Errorf("tags Object missing name: %w", ParseError))
					return
				}
				tags.addTag(strVal)
				tagValue, _ := tags.GetTag(strVal)
				objStr, dType, _, err := jsonparser.Get(value, "values")
				if dType != jsonparser.Object || err != nil {
					collector.Add(fmt.Errorf("tags Object missing value object: %w", ParseError))
					return
				}
				jsonparser.ObjectEach(objStr, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
					if dataType != jsonparser.String && dataType != jsonparser.Number {
						collector.Add(fmt.Errorf("tags Object key '%s' has invalid type '%s': %w", key, dataType, ParseError))
						return nil
					}

					//warning: this code expect that value is string representation of number
					//if dataType is jsonparser.Number
					tagValue.Put(string(key), string(value))

					return nil
				})
				//tagValue.freeze() make it after spawn
			default:
				return
			}
		})
	case jsonparser.Object:
		jsonparser.ObjectEach(tagsCfg, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			switch dataType {
			case jsonparser.Boolean:
				bVal, err := jsonparser.GetBoolean(value)
				if !collector.Add(err) && bVal {
					tags.addTag(string(key))
				}
			case jsonparser.Object:
				tags.addTag(string(key))
				tagValue, _ := tags.GetTag(string(key))
				jsonparser.ObjectEach(value, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
					if dataType != jsonparser.String && dataType != jsonparser.Number {
						collector.Add(fmt.Errorf("tags Object key '%s' has invalid type '%s': %w", key, dataType, ParseError))
						return nil
					}
					//warning: this code expect that value is string representation of number
					//if dataType is jsonparser.Number
					tagValue.Put(string(key), string(value))
					return nil
				})
				//tagValue.freeze() make it after spawn
			default:
				collector.Add(fmt.Errorf("tags Object key '%s' has invalid type '%s': %w", key, dataType, ParseError))
			}
			return nil
		})
	case jsonparser.Null:
		fallthrough
	case jsonparser.NotExist:
		//nope
	default:
		collector.Add(fmt.Errorf("tags: %w", ParseError))
	}

	return tags
}

func CollisionLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		collision *collider.ClBody
	)

	x, err := jsonparser.GetFloat(payload, "x")
	collector.Add(err)
	y, err := jsonparser.GetFloat(payload, "y")
	collector.Add(err)
	w, err := jsonparser.GetFloat(payload, "w")
	collector.Add(err)
	h, err := jsonparser.GetFloat(payload, "h")
	collector.Add(err)

	if x == y && x == w && x == h && x == 0 {
		collector.Add(errors.New("zero dimension collision not allowed"))
		return nil
	}

	cType, _ := jsonparser.GetString(payload, "type")
	switch cType {
	case "static":
		collision = collider.NewStaticCollision(x, y, w, h)
	case "penetrate":
		collision = collider.NewPenetrateCollision(x, y, w, h)
	case "nocollision":
		collision = collider.NewFakeCollision(x, y, w, h)
	case "vision":
		collision = collider.NewPenetrateCollision(x, y, w, h)
	case "perimeter":
		collision = collider.NewPerimeterCollision(x, y, w, h)
	default:
		collision = collider.NewCollision(x, y, w, h)
	}

	return collision
}

func SpriterLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		sprite     Spriteer
		spriteConf SpriteerConfig
		ok         bool
	)

	if spriteConf, ok = preset.(SpriteerConfig); !ok {
		spriteConf = SpriteerConfig{}
	} else {
		if _, dt, _, _ := jsonparser.Get(payload, "custom"); dt == jsonparser.Object {
			spriteConf.Custom = make(map[string]int)
		}
	}
	collector.Add(json.Unmarshal(payload, &spriteConf))

	switch spriteConf.Type {
	case "animation":
		if obj, err := lGetObject(ctx, spriteConf.Type, get, collector, AnimationConfig{
			SpriteerConfig: spriteConf,
		}, payload); !collector.Add(err) {
			sprite = obj.(Spriteer)
		}
	case "composition":
		if obj, err := lGetObject(ctx, spriteConf.Type, get, collector, CompositionConfig{
			SpriteerConfig: spriteConf,
		}, payload); !collector.Add(err) {
			sprite = obj.(Spriteer)
		}
	default:
		if obj, err := lGetObject(ctx, spriteConf.Type, get, collector, SpriteConfig{
			SpriteerConfig: spriteConf,
		}, payload); !collector.Add(err) {
			sprite = obj.(Spriteer)
		}
	}

	if sprite == nil {
		collector.Add(fmt.Errorf("sprite is nil, use default: %w", InstanceError))
		sprite = ErrorSprite
	}

	return sprite
}

func SpriteLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		sprite     *Sprite
		spriteConf SpriteConfig
		ok         bool
		cfg        *GameConfig
		err        error
	)

	if obj, err := lGetObject(ctx, "gameConfig", get, collector, preset, payload); !collector.Add(err) {
		cfg = obj.(*GameConfig)
	}

	if spriteConf, ok = preset.(SpriteConfig); !ok {
		spriteConf = SpriteConfig{}
	}
	collector.Add(json.Unmarshal(payload, &spriteConf))

	if spriteConf.Name == "" {
		collector.Add(fmt.Errorf("sprite must have a id: %w", ParseError))
		return ErrorSprite
	}

	sprite, err = GetSprite(spriteConf.Name, true, spriteConf.IsTransparent)
	if !collector.Add(err) {
		sprite.CalculateSize()
		if len(spriteConf.Custom) > 0 && !cfg.disableCustomization {
			hash := hashCustomizeMap(spriteConf.Custom)
			customSprite, err := GetSprite(spriteConf.Name+"-"+hash, false, false)
			if err == nil {
				sprite = customSprite
			} else {
				sprite, err = CustomizeSprite(sprite, spriteConf.Custom)
				if !collector.Add(err) {
					err = AddSprite(spriteConf.Name+"-"+hash, sprite)
				}
			}
			collector.Add(err)
		}
	}

	if sprite == nil {
		collector.Add(fmt.Errorf("sprite is nil, use default: %w", InstanceError))
		sprite = ErrorSprite
	}

	return sprite
}

func CompositionLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		composition *Composition
		layerConf   *CompositionLayerConfig
	)
	//skip error because of dataType validation
	keyFrames, dataType, _, _ := jsonparser.Get(payload, "frames")
	switch dataType {
	case jsonparser.Array:
		index := 0
		jsonparser.ArrayEach(keyFrames, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			collector.tracePush("[" + strconv.Itoa(index) + "]")
			switch dataType {
			case jsonparser.String:
				frame, err := GetSprite(string(value), true, false)
				if !collector.Add(err) {
					composition.addFrame(frame, 0, 0, index)
				}
			case jsonparser.Object:
				if frame, err := lGetObject(ctx, "spriter", get, collector, preset, value); !collector.Add(err) {
					if cfg, ok := preset.(*CompositionLayerConfig); ok { //todo separate defaults and params
						layerConf = cfg
					} else {
						layerConf = new(CompositionLayerConfig)
					}
					json.Unmarshal(value, layerConf)
					if layerConf.ZIndex == 0 {
						_, dataType, _, _ = jsonparser.Get(value, "zIndex")
						if dataType == jsonparser.NotExist || dataType == jsonparser.Null {
							layerConf.ZIndex = index
						}
					}
					composition.addFrame(frame.(Spriteer), layerConf.OffsetX, layerConf.OffsetY, layerConf.ZIndex)
				}
			default:
				collector.Add(fmt.Errorf("keyFrames:  %w", ParseError))
			}
			collector.tracePop()
			index++
		})
	default:
		collector.Add(fmt.Errorf("frames: %w", ParseError))
	}

	return composition
}

func AnimationLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		animation *Animation
		cfg       *GameConfig
		config    AnimationConfig
		ok        bool
		err       error
	)

	if obj, err := lGetObject(ctx, "gameConfig", get, collector, preset, payload); !collector.Add(err) {
		cfg = obj.(*GameConfig)
	}

	if config, ok = preset.(AnimationConfig); !ok {
		config = AnimationConfig{}
	}

	err = json.Unmarshal(payload, &config)
	if err != nil {
		collector.Add(fmt.Errorf("animation config deserialization error: %w", ParseError))
		return ErrorAnimation
	}

	if config.Length <= 0 {
		collector.Add(fmt.Errorf("animation %s length must be > 0: %w", config.Name, ParseError))
		return ErrorAnimation
	} else if config.Name == "" {
		collector.Add(fmt.Errorf("Name must be set: %w", ParseError))
		return ErrorAnimation
	}

	if config.Path == "" {
		config.Path = config.Name
	}

	//cache only call
	animation, err = GetAnimation2(config.Name)

	if err != nil {
		//skip error because of dataType validation
		keyFrames, dataType, _, _ := jsonparser.Get(payload, "keyFrames")

		switch dataType {
		case jsonparser.Array:
			animation, _ = NewAnimation(nil)
			index := 0
			jsonparser.ArrayEach(keyFrames, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				collector.tracePush("[" + strconv.Itoa(index) + "]")
				switch dataType {
				case jsonparser.String:
					frame, err := GetSprite(string(value), true, config.IsTransparent)
					if !collector.Add(err) {
						err = animation.AddFrame(frame)
						if collector.Add(err) && errors.Is(err, FrameTypeCombinationError) {
							return
						}
					} else {
						animation.AddFrame(ErrorSprite)
					}
				case jsonparser.Object:
					if obj, err := lGetObject(ctx, "spriter", get, collector, preset, value); err == nil {
						err = animation.AddFrame(obj.(Spriteer))
						if collector.Add(err) && errors.Is(err, FrameTypeCombinationError) {
							return
						}
					} else {
						collector.Add(fmt.Errorf("animation no frame: %s", InstanceError))
						animation.AddFrame(ErrorSprite)
					}
				default:
					collector.Add(fmt.Errorf("keyFrames:  %w", ParseError))
				}
				collector.tracePop()
				index++
			})
			if index != config.Length {
				collector.Add(fmt.Errorf("length != len(keyFrames) %d, %d:  %w", len(keyFrames), config.Length, ParseError))
				return ErrorAnimation
			}
			err = AddAnimation(config.Name, animation)
			collector.Add(err)
			animation = animation.Copy()
		case jsonparser.Null:
			fallthrough
		case jsonparser.NotExist:
			animation, err = LoadAnimation2(config.Path, config.Length, config.IsTransparent)
			if !collector.Add(err) {
				err = AddAnimation(config.Name, animation)
				collector.Add(err)
				animation = animation.Copy()
			}
		default:
			collector.Add(fmt.Errorf("keyFrames: %w", ParseError))
		}
	}

	if animation != ErrorAnimation {
		//todo there is a problem, config.name is kind a animation key but to simplify usage we allow to override animation base config
		if len(config.Custom) > 0 && !cfg.disableCustomization {
			name := hashCustomizeMap(config.Custom)
			if customized, err := GetAnimation2(config.Name + "-" + name); err != nil {
				customized, err = CustomizeAnimation(animation, config.Name, config.Custom)
				if !collector.Add(err) {
					err = AddAnimation(config.Name+"-"+name, customized)
					collector.Add(err)
					animation = customized.Copy()
				}
			} else {
				animation = customized
			}
		}

		animation.Cycled = config.Cycled
		animation.Duration = config.Duration
		if config.Blink <= 0 {
			animation.BlinkRate = -1
		} else {
			animation.BlinkRate = config.Blink
		}
		if config.RepeatDuration <= 0 {
			animation.RepeatDuration = -1
		} else {
			animation.RepeatDuration = config.RepeatDuration
		}
		animation.Reversed = config.Reversed
		if animation.Duration == 0 && animation.collection {
			collector.Add(fmt.Errorf("Duration is zero: %w", ParseError))
		}
	}

	return animation
}

func StateLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {

	var (
		state *State
	)

	if root, err := lGetObject(ctx, "stateItem", get, collector, preset, payload); !collector.Add(err) {
		state, _ = NewState(nil)
		state.root = root.(*StateItem)
		state.Current = root.(*StateItem)
		state.defaultPath, err = jsonparser.GetString(payload, "default")
		collector.Add(err)
		if state.defaultPath == ToDefaultState {
			collector.Add(fmt.Errorf("%s is predefined state value and it cant be a defaultPath", ToDefaultState))
			state.defaultPath = "/"
		}
		state.MoveTo(state.defaultPath)
	}

	return state
}

func StateItemLoader(ctx context.Context, get LoaderGetter, collector *LoadErrors, preset interface{}, payload []byte) interface{} {
	var (
		sprite     Spriteer
		collision  *collider.ClBody
		x, y, w, h float64
		err        error
	)

	if customBytes, dt, _, _ := jsonparser.Get(payload, "custom"); dt == jsonparser.Object {
		if spriteConf, ok := preset.(SpriteerConfig); ok {
			spriteConf.Custom = make(map[string]int)
			if !collector.Add(json.Unmarshal(customBytes, &spriteConf)) {
				preset = spriteConf
			}
		}
	}

	//compatibility
	//skip error because of dataType validation
	compability := false
	spriteCfg, dType, _, _ := jsonparser.Get(payload, "animation")
	if dType == jsonparser.NotExist || dType == jsonparser.Null {
		spriteCfg, dType, _, _ = jsonparser.Get(payload, "sprite")
	} else {
		if dType != jsonparser.NotExist {
			collector.Add(fmt.Errorf("animation key is depricated for jsonLoaders: %w", ParseError))
		}
		compability = true
	}

	switch dType {
	case jsonparser.String:
		if compability {
			sprite, err = ErrorAnimation, fmt.Errorf("string declaration in compability mode: %w", ParseError)
		} else {
			sprite, err = GetSprite(string(spriteCfg), true, false)
		}
		collector.Add(err)
	case jsonparser.Object:
		var blueprint string
		if compability {
			blueprint = "animation"
		} else {
			blueprint = "spriter"
		}
		if obj, err := lGetObject(ctx, blueprint, get, collector, preset, spriteCfg); !collector.Add(err) {
			sprite = obj.(Spriteer)
		}
	default:
		collector.Add(fmt.Errorf("animation: %w", ParseError))
	}

	//skip error because of dataType validation
	collisionCfg, dType, _, _ := jsonparser.Get(payload, "collision")
	switch dType {
	case jsonparser.Object:
		if obj, err := lGetObject(ctx, "collision", get, collector, preset, collisionCfg); !collector.Add(err) {
			collision = obj.(*collider.ClBody)
		}
	case jsonparser.Null:
		//nope for now
	case jsonparser.NotExist:
		//nope for now
	default:
		collector.Add(fmt.Errorf("collision: %w", ParseError))
	}

	//temporal
	if collision != nil {
		x, y, w, h = collision.GetRect()
	}

	parent, err := NewStateItem(nil, &UnitStateInfo{
		sprite:     sprite,
		collisionX: x,
		collisionY: y,
		collisionW: w,
		collisionH: h,
	})
	collector.Add(err)

	//skip error because of dataType validation
	items, dataType, _, _ := jsonparser.Get(payload, "items")
	switch dataType {
	case jsonparser.Object:
		index := 0
		jsonparser.ObjectEach(items, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			collector.tracePush("[" + strconv.Itoa(index) + "]")
			if string(key) == ToDefaultState {
				collector.Add(fmt.Errorf("%s is predefined state value and it cant be a stateName", ToDefaultState))
				//todo return ErrorStateItem
			}
			if state := StateItemLoader(ctx, get, collector, preset, value); state != nil {
				state.(*StateItem).parent = parent
				parent.items[string(key)] = state.(*StateItem)
			}
			collector.tracePop()
			index++
			return nil
		})
	case jsonparser.Null:
		//nope for now
	case jsonparser.NotExist:
		//nope for now
	default:
		collector.Add(fmt.Errorf("items: %w", ParseError))
	}

	return parent
}

func arrayLength(array []byte, keys ...string) int {
	index := 0
	jsonparser.ArrayEach(array, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		index++
	}, keys...)
	return index
}
