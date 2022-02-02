package main

type Attributes struct {
	ID          int64
	Team        int8
	TeamTag     string
	Type        string
	Blueprint   string
	Name        string
	Description string
	Layer       int
	Require     []string
	Player      bool
	Obstacle    bool
	Danger      bool
	Vulnerable  bool
	Motioner    bool
	Evented     bool
	Controled   bool
	Collided    bool
	Visioned    bool
	Renderable  bool
	Tagable     bool
	AI          bool
	Spawned     bool
	Destroyed   bool
	Custom      CustomizeMap
}
