package main

type Danger interface {
	GetDamage(target Vulnerable) (value int, nemesis ObjectInterface)
	HasTag(tag string) bool
}

type Vulnerable interface {
	ReciveDamage(incoming Danger)
	HasTag(tag string) bool
}

type Endurance struct {
	HP, FullHP int
}
