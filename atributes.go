package main

type Attributes struct {
	ID              int64
	Team 			int8
	TeamTag			string
	Blueprint       string
	Player			bool
	Obstacle 		bool
	Danger          bool
	Vulnerable		bool
	Motioner		bool
	Evented			bool
	Controled		bool
	Collided		bool
	Renderable		bool
	Tagable			bool
}
