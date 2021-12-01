package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
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
	instance.M["gun"] = GunLoader
	instance.M["motionObject"] = MotionObjectLoader
	instance.M["object"] = ObjectLoader
	instance.M["state"] = StateLoader
	instance.M["stateItem"] = StateItemLoader
	instance.M["collision"] = CollisionLoader
	instance.M["sprite"] = SpriteLoader
	instance.M["animation"] = AnimationLoader
	instance.M["composition"] = CompositionLoader

	return instance
}

func RootLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	uType, err := jsonparser.GetString(payload, "type")
	if collector.Add(err) {
		return nil
	}
	loader := get(uType)
	if loader == nil {
		collector.Add(fmt.Errorf("%s: %w", uType, LoaderNotFoundError))
		return nil
	}
	object := loader(get, collector, payload)
	if object == nil {
		return nil
	}
	object.(ObjectInterface).GetAttr().Blueprint = uType

	return object
}

func UnitLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		output    EventChanel
		motionObj *MotionObject
		stateObj  *State
		oo        *ObservableObject
		co        *ControlledObject
		unit      *Unit
		gun       *Gun
		control   *controller.Control
		err       error
	)

	if loader := get("motionObject"); loader != nil {
		if moObj := loader(get, collector, payload); moObj != nil {
			motionObj = moObj.(*MotionObject)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "motionObject", LoaderNotFoundError))
	}
	//skip error because of dataType validation
	stateCfg, dType, _, err := jsonparser.Get(payload, "state")
	switch dType {
	case jsonparser.String:
		//todo rename
		stateObj, err = GetTankState(string(stateCfg))
		collector.Add(err)
	case jsonparser.Object:
		if loader := get("state"); loader != nil {
			if stObj := loader(get, collector, stateCfg); stObj != nil {
				stateObj = stObj.(*State)
			}
		} else {
			collector.Add(fmt.Errorf("%s: %w", "state", LoaderNotFoundError))
		}
	default:
		collector.Add(fmt.Errorf("state: %w", ParseError))
	}

	if motionObj == nil {
		return nil
	}

	//skip error because of dataType validation
	gunCfg, dType, _, _ := jsonparser.Get(payload, "gun")
	switch dType {
	case jsonparser.Object:
		if loader := get("gun"); loader != nil {
			if gunObj := loader(get, collector, gunCfg); gunObj != nil {
				gun = gunObj.(*Gun)
			}
		} else {
			collector.Add(fmt.Errorf("%s: %w", "gun", LoaderNotFoundError))
		}
	default:
		collector.Add(fmt.Errorf("gun: %w", ParseError))
	}

	if loader := get("eventChanel"); loader != nil {
		if outObj := loader(get, collector, payload); outObj != nil {
			output = outObj.(EventChanel)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "eventChanel", LoaderNotFoundError))
	}

	oo, err = NewObservableObject(output, nil)
	if !collector.Add(err) {
		//skip error because of dataType validation
		_, dataType, _, _ := jsonparser.Get(payload, "control")
		switch dataType {
		case jsonparser.Null:
			control = nil
		default:
			if DEBUG_NO_AI {
				control, _ = controller.NewNoneControl()
			} else {
				control, _ = controller.NewAIControl()
			}
		}
		co, err = NewControlledObject(control, nil)
		if !collector.Add(err) {
			unit, err = NewUnit2(co, oo, motionObj, stateObj)
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

		hp, err := jsonparser.GetInt(payload, "hp")
		if !collector.Add(err) {
			unit.FullHP = int(hp)
			unit.HP = int(hp)
		}

		score, err := jsonparser.GetInt(payload, "score")
		if !collector.Add(err) {
			unit.Score = int(score)
		}
	}

	return unit
}

func WallLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		object   *Object
		stateObj *State
		oo       *ObservableObject
		wall     *Wall
		output   EventChanel
		err      error
	)

	if loader := get("object"); loader != nil {
		if obj := loader(get, collector, payload); obj != nil {
			object = obj.(*Object)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "object", LoaderNotFoundError))
	}

	//skip error because of dataType validation
	stateCfg, dType, _, _ := jsonparser.Get(payload, "state")
	switch dType {
	case jsonparser.String:
		//todo rename
		stateObj, err = GetTankState(string(stateCfg))
		collector.Add(err)
	case jsonparser.Object:
		if loader := get("state"); loader != nil {
			if stObj := loader(get, collector, stateCfg); stObj != nil {
				stateObj = stObj.(*State)
			}
		} else {
			collector.Add(fmt.Errorf("%s: %w", "state", LoaderNotFoundError))
		}
	default:
		collector.Add(fmt.Errorf("state: %w", ParseError))
	}

	if object == nil {
		return nil
	}

	if loader := get("eventChanel"); loader != nil {
		if outObj := loader(get, collector, payload); outObj != nil {
			output = outObj.(EventChanel)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "eventChanel", LoaderNotFoundError))
	}

	oo, err = NewObservableObject(output, nil)

	if !collector.Add(err) {
		wall, err = NewWall2(*object, stateObj, oo)
		collector.Add(err)
	}

	if wall != nil {
		wall.Attributes.Obstacle = true
		wall.Attributes.Vulnerable = true
		wall.Attributes.Evented = true

		hp, err := jsonparser.GetInt(payload, "hp")
		if !collector.Add(err) {
			wall.FullHP = int(hp)
			wall.HP = int(hp)
		}

		score, err := jsonparser.GetInt(payload, "score")
		if !collector.Add(err) {
			wall.Score = int(score)
		}
	}

	return wall
}

func CollectableLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		object   *Object
		stateObj *State
		oo       *ObservableObject
		collect  *Collectable
		output   EventChanel
		err      error
	)

	if loader := get("object"); loader != nil {
		if obj := loader(get, collector, payload); obj != nil {
			object = obj.(*Object)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "object", LoaderNotFoundError))
	}

	//skip error because of dataType validation
	stateCfg, dType, _, _ := jsonparser.Get(payload, "state")
	switch dType {
	case jsonparser.String:
		//todo rename
		stateObj, err = GetTankState(string(stateCfg))
		collector.Add(err)
	case jsonparser.Object:
		if loader := get("state"); loader != nil {
			if stObj := loader(get, collector, stateCfg); stObj != nil {
				stateObj = stObj.(*State)
			}
		} else {
			collector.Add(fmt.Errorf("%s: %w", "state", LoaderNotFoundError))
		}
	default:
		collector.Add(fmt.Errorf("state: %w", ParseError))
	}

	if object == nil {
		return nil
	}

	if loader := get("eventChanel"); loader != nil {
		if outObj := loader(get, collector, payload); outObj != nil {
			output = outObj.(EventChanel)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "eventChanel", LoaderNotFoundError))
	}

	oo, err = NewObservableObject(output, nil)

	if !collector.Add(err) {
		collect, err = NewCollectable2(object, oo, stateObj, nil)
		collector.Add(err)
	}

	if collect != nil {
		collect.Attributes.Obstacle = true
		collect.Attributes.Evented = true

		ttl, err := jsonparser.GetInt(payload, "ttl")
		if !collector.Add(err) {
			collect.Ttl = time.Duration(ttl)
		}
	}

	return collect
}

func ExplosionLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		object *Object
		/*		stateObj  	*State*/
		oo        *ObservableObject
		explosion *Explosion
		output    EventChanel
		err       error
	)

	if loader := get("object"); loader != nil {
		if obj := loader(get, collector, payload); obj != nil {
			object = obj.(*Object)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "object", LoaderNotFoundError))
	}

	/*	stateCfg, dType, _, err := jsonparser.Get(payload, "state")
		switch dType {
		case jsonparser.String:
			//todo rename
			stateObj, err = GetTankState(string(stateCfg))
			collector.Add(err)
		case jsonparser.Object:
			if loader := get("state"); loader != nil {
				if stObj := loader(get, collector, stateCfg); stObj != nil {
					stateObj = stObj.(*State)
				}
			} else {
				collector.Add(fmt.Errorf("%s: %w", "state", LoaderNotFoundError))
			}
		default:
			collector.Add(fmt.Errorf("state: %w", ParseError))
		}*/

	if object == nil {
		return nil
	}

	if loader := get("eventChanel"); loader != nil {
		if outObj := loader(get, collector, payload); outObj != nil {
			output = outObj.(EventChanel)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "eventChanel", LoaderNotFoundError))
	}

	oo, err = NewObservableObject(output, nil)

	if !collector.Add(err) {
		explosion, err = NewExplosion2(object, oo, nil)
		collector.Add(err)
	}

	if explosion != nil {
		explosion.Attributes.Evented = true
		explosion.Attributes.Danger = true

		ttl, err := jsonparser.GetInt(payload, "ttl")
		if !collector.Add(err) {
			explosion.Ttl = time.Duration(ttl)
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

func ProjectileLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		motionObj  *MotionObject
		stateObj   *State
		oo         *ObservableObject
		projectile *Projectile
		output     EventChanel
		err        error
	)

	if loader := get("motionObject"); loader != nil {
		if moObj := loader(get, collector, payload); moObj != nil {
			motionObj = moObj.(*MotionObject)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "motionObject", LoaderNotFoundError))
	}

	//skip error because of dataType validation
	stateCfg, dType, _, _ := jsonparser.Get(payload, "state")
	switch dType {
	case jsonparser.String:
		//todo rename
		stateObj, err = GetTankState(string(stateCfg))
		collector.Add(err)
	case jsonparser.Object:
		if loader := get("state"); loader != nil {
			if stObj := loader(get, collector, stateCfg); stObj != nil {
				stateObj = stObj.(*State)
			}
		} else {
			collector.Add(fmt.Errorf("%s: %w", "state", LoaderNotFoundError))
		}
	default:
		collector.Add(fmt.Errorf("state: %w", ParseError))
	}

	if motionObj == nil {
		return nil
	}

	if loader := get("eventChanel"); loader != nil {
		if outObj := loader(get, collector, payload); outObj != nil {
			output = outObj.(EventChanel)
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "eventChanel", LoaderNotFoundError))
	}

	oo, err = NewObservableObject(output, nil)

	if !collector.Add(err) {
		projectile, err = NewProjectile2(motionObj, oo, stateObj, nil)
		if !collector.Add(err) {

		}
	}

	if projectile != nil {
		projectile.Attributes.Obstacle = true
		projectile.Attributes.Vulnerable = true
		projectile.Attributes.Motioner = true
		projectile.Attributes.Evented = true
		projectile.Attributes.Controled = true

		ttl, err := jsonparser.GetInt(payload, "ttl")
		if !collector.Add(err) {
			projectile.Ttl = time.Duration(ttl)
		}

		damage, err := jsonparser.GetInt(payload, "damage")
		if !collector.Add(err) {
			projectile.Damage = int(damage)
		}
	}

	return projectile
}

func GunLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
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

func MotionObjectLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		object *MotionObject
		err    error
	)

	if loader := get("object"); loader != nil {
		if obj := loader(get, collector, payload); obj != nil {
			config := new(MotionObjectConfig2)
			if !collector.Add(json.Unmarshal(payload, config)) {
				if config.Direction.X == 0 && config.Direction.Y == 0 {
					config.Direction.Y = -1
				}
				_, spdMin, _, _ := jsonparser.Get(payload, "speed", "min")
				_, spdMax, _, _ := jsonparser.Get(payload, "speed", "max")
				object, err = NewMotionObject2(obj.(*Object), config.Direction, Point{
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
		} else {
			return nil
		}
	} else {
		collector.Add(fmt.Errorf("%s: %w", "object", LoaderNotFoundError))
	}

	if object != nil {
		object.Attributes.Motioner = true
	}

	return object
}

func ObjectLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		sprite    Spriteer
		collision *collider.ClBody
		err       error
	)

	//skip error because of dataType validation
	spriteCfg, dType, _, _ := jsonparser.Get(payload, "sprite")
	switch dType {
	case jsonparser.String:
		sprite, err = GetSprite(string(spriteCfg), true, false)
		collector.Add(err)
	case jsonparser.Object:
		loader := get("sprite")
		if loader != nil {
			proxy := loader(get, collector, spriteCfg)
			if proxy != nil {
				sprite = proxy.(Spriteer)
			}
		} else {
			collector.Add(fmt.Errorf("sprite: %w", LoaderNotFoundError))
		}
	default:
		collector.Add(fmt.Errorf("sprite: %w", ParseError))
	}

	//skip error because of dataType validation
	collisionCfg, dType, _, _ := jsonparser.Get(payload, "collision")
	switch dType {
	case jsonparser.Object:
		loader := get("collision")
		if loader != nil {
			proxy := loader(get, collector, collisionCfg)
			if proxy != nil {
				collision = proxy.(*collider.ClBody)
			}
		} else {
			collector.Add(fmt.Errorf("collision: %w", LoaderNotFoundError))
		}
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

		//skip error because of dataType validation
		tagsCfg, dType, _, _ := jsonparser.Get(payload, "tags")
		switch dType {
		case jsonparser.Array:
			jsonparser.ArrayEach(tagsCfg, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				switch dataType {
				case jsonparser.String:
					object.addTag(string(value))
				case jsonparser.Object:
					strVal, err := jsonparser.GetString(value, "name")
					if err != nil || strVal == "" {
						collector.Add(fmt.Errorf("tags Object missing name: %w", ParseError))
						return
					}
					object.addTag(strVal)
					tagValue, _ := object.GetTag(strVal, true)
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
					object.addTag(string(key))
				case jsonparser.Object:
					tagValue, _ := object.GetTag(string(key), true)
					object.addTag(string(key))
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

		zIndex, err := jsonparser.GetInt(payload, "zIndex")
		if err != nil {
			object.zIndex = int(zIndex)
		}
	}

	return object
}

func CollisionLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
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
	default:
		collision = collider.NewCollision(x, y, w, h)
	}

	return collision
}

func SpriteLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		sprite Spriteer
		err    error
	)

	sType, _ := jsonparser.GetString(payload, "type")
	switch sType {
	case "animation":
		fallthrough
	case "composition":
		loader := get(sType)
		if loader != nil {
			proxy := loader(get, collector, payload)
			if proxy != nil {
				sprite = proxy.(Spriteer)
			}
		} else {
			collector.Add(fmt.Errorf("%s: %w", sType, LoaderNotFoundError))
		}
	default:
		var isTransparent bool
		sId, _ := jsonparser.GetString(payload, "name")
		isTransparent, err = jsonparser.GetBoolean(payload, "transparent")
		if err != nil {
			isTransparent = false
		}
		if sId != "" {
			var sprite *Sprite
			sprite, err = GetSprite(sId, true, isTransparent)
			if !collector.Add(err) {
				collector.Add(fmt.Errorf("sprite: %w", ParseError))
				customBytes, dataType, _, _ := jsonparser.Get(payload, "custom")
				switch dataType {
				case jsonparser.Object:
					custom := make(CustomizeMap)
					if !collector.Add(json.Unmarshal(customBytes, &custom)) {
						hash := hashCustomizeMap(custom)
						customSprite, err := GetSprite(sId+"-"+hash, false, false)
						if err != nil {
							collector.Add(err)
							sprite = customSprite
						} else {
							sprite, err = CustomizeSprite(sprite, custom)
							if !collector.Add(err) {
								sprite, err = AddSprite(sId+"-"+hash, sprite)
							}
						}
						collector.Add(err)
					}
				default:
					collector.Add(fmt.Errorf("custom: %w", ParseError))
				}

			}
		} else {
			collector.Add(fmt.Errorf("sprite must have a id: %w", ParseError))
		}
	}

	if sprite == nil {
		collector.Add(fmt.Errorf("sprite is nil, use default: %w", InstanceError))
		sprite = ErrorSprite
	}

	return sprite
}

func CompositionLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		composition *Composition
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
				if loader := get("sprite"); loader == nil {
					collector.Add(fmt.Errorf("%s: %w", "sprite", LoaderNotFoundError))
				} else {
					if frame := loader(get, collector, value); frame != nil {
						layerConf := new(CompositionLayerConfig)
						json.Unmarshal(value, layerConf)
						if layerConf.ZIndex == 0 {
							_, dataType, _, _ = jsonparser.Get(value, "zIndex")
							if dataType == jsonparser.NotExist || dataType == jsonparser.Null {
								layerConf.ZIndex = index
							}
						}
						composition.addFrame(frame.(Spriteer), layerConf.OffsetX, layerConf.OffsetY, layerConf.ZIndex)
					}
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

func AnimationLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		animation *Animation
		err       error
	)

	config := new(AnimationConfig)
	err = json.Unmarshal(payload, config)

	if err != nil {
		collector.Add(fmt.Errorf("animation config deserialization error: %w", ParseError))
		return ErrorAnimation
	}

	if config.Length <= 0 {
		collector.Add(fmt.Errorf("animation %s length must be > 0: %w", config.Name, ParseError))
		return ErrorAnimation
	} else if config.Name == "" {
		collector.Add(fmt.Errorf("name must be set: %w", ParseError))
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
					if loader := get("sprite"); loader == nil {
						collector.Add(fmt.Errorf("%s: %w", "sprite", LoaderNotFoundError))
					} else {
						if frame := loader(get, collector, value); frame != nil {
							err = animation.AddFrame(frame.(Spriteer))
							if collector.Add(err) && errors.Is(err, FrameTypeCombinationError) {
								return
							}
						} else {
							collector.Add(fmt.Errorf("animation no frame: %s", InstanceError))
							animation.AddFrame(ErrorSprite)
						}
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
			AddAnimation(config.Name, animation)
			animation = animation.Copy()
		case jsonparser.Null:
			fallthrough
		case jsonparser.NotExist:
			animation, err = LoadAnimation2(config.Path, config.Length, config.IsTransparent)
			if !collector.Add(err) {
				AddAnimation(config.Name, animation)
				animation = animation.Copy()
			}
		default:
			collector.Add(fmt.Errorf("keyFrames: %w", ParseError))
		}
	}

	if animation != ErrorAnimation {
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
		if config.Custom != nil {
			customizeSliceSprite(animation.keyFrames, config.Name, config.Custom, collector)
		}
	}

	return animation
}

func StateLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {

	var (
		loader Loader
		state  *State
		err    error
	)

	if loader = get("stateItem"); loader == nil {
		collector.Add(fmt.Errorf("stateItem: %w", LoaderNotFoundError))
	} else {
		if root := loader(get, collector, payload); root != nil {
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
	}

	return state
}

func StateItemLoader(get LoaderGetter, collector *LoadErrors, payload []byte) interface{} {
	var (
		sprite     Spriteer
		collision  *collider.ClBody
		x, y, w, h float64
		err        error
	)

	//compatibility
	//skip error because of dataType validation
	spriteCfg, dType, _, _ := jsonparser.Get(payload, "animation")
	if dType == jsonparser.NotExist || dType == jsonparser.Null {
		spriteCfg, dType, _, _ = jsonparser.Get(payload, "sprite")
	}

	switch dType {
	case jsonparser.String:
		sprite, err = GetSprite(string(spriteCfg), true, false)
		collector.Add(err)
	case jsonparser.Object:
		loader := get("sprite")
		if loader != nil {
			proxy := loader(get, collector, spriteCfg)
			if proxy != nil {
				sprite = proxy.(Spriteer)
			}
		} else {
			collector.Add(fmt.Errorf("sprite: %w", LoaderNotFoundError))
		}
	default:
		collector.Add(fmt.Errorf("animation: %w", ParseError))
	}

	//skip error because of dataType validation
	collisionCfg, dType, _, _ := jsonparser.Get(payload, "collision")
	switch dType {
	case jsonparser.Object:
		loader := get("collision")
		if loader != nil {
			proxy := loader(get, collector, collisionCfg)
			if proxy != nil {
				collision = proxy.(*collider.ClBody)
			}
		} else {
			collector.Add(fmt.Errorf("collision: %w", LoaderNotFoundError))
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
			if state := StateItemLoader(get, collector, value); state != nil {
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

func customizeSliceSprite(sprites []Spriteer, name string, custom CustomizeMap, collector *LoadErrors) {
	for i, frame := range sprites {
		if s, ok := frame.(*Sprite); ok {
			if !IsCustomizedSpriteVer(s) {
				if frameCustom, err := GetSprite(customizedSpriteName(name, custom), false, false); frameCustom != nil {
					collector.Add(err)
					sprites[i] = frameCustom
				} else {
					frameCustom, err := CustomizeSprite(s, custom)
					if !collector.Add(err) {
						frameCustom, err = AddSprite(customizedSpriteName(name, custom), frameCustom)
						collector.Add(err)
						sprites[i] = frameCustom
					}
				}
			}
		}
	}
}
