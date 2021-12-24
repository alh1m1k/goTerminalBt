package main

import (
	"GoConsoleBT/collider"
	pathfinding "github.com/xarg/gopathfinding"
	"sync"
	"time"
)

type PathReceiver interface {
	ReceivePath(path []Zone, jobId int64)
}

const (
	NJ_STATE_NEW int = iota - 1
	NJ_STATE_WORK
	NJ_STATE_DONE
)

var (
	NoZone = Zone{
		X: -100,
		Y: -100,
	}
)

type NavJob struct {
	jobId  int64
	input  []*Tracker
	from   Zone
	to     Zone
	output []Zone
	state  int
	owner  PathReceiver
}

type Navigation struct {
	*Location
	*collider.Collider
	queue           []*NavJob
	mutex           sync.Mutex
	mapDataTemplate pathfinding.MapData
	NavData         [][]Zone
}

func (receiver *Navigation) Execute(timeLeft time.Duration) {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	if len(receiver.queue) > 0 {
		mapData, err := receiver.Location.Mapdata()
		if err == nil {
			emptyUntil := -1
			for index, job := range receiver.queue {
				if job != nil && emptyUntil == -1 {
					emptyUntil = index
				}
				if job == nil {
					continue
				}
				if job.state == NJ_STATE_DONE {
					//notify object: plan ready
					if job.output != nil && len(job.output) > 0 {
						receiver.NavData = append(receiver.NavData, job.output)
					}
					if job.owner != nil {
						go job.owner.ReceivePath(job.output, job.jobId)
					}
					receiver.queue[index] = nil
				} else if job.state == NJ_STATE_NEW {
					job.state = NJ_STATE_WORK
					job.input = mapData
					go receiver.buildPath(job)
				}
			}
			if emptyUntil > 0 {
				//receiver.queue = receiver.queue[emptyUntil:]
				//todo implement compact
			}
		} else {
			logger.Printf("error acquire map data\n")
		}
	}
}

func (receiver *Navigation) buildPath(job *NavJob) error {
	mapData := make(pathfinding.MapData, len(receiver.mapDataTemplate))
	copy(mapData, receiver.mapDataTemplate)
	for i, _ := range receiver.mapDataTemplate {
		mapData[i] = make([]int, len(receiver.mapDataTemplate[i]))
		copy(mapData[i], receiver.mapDataTemplate[i])
	}
	for _, track := range job.input {
		x, y := track.GetIndexes()
		mapData[y][x] = pathfinding.WALL
	}
	mapData[job.from.Y][job.from.X] = pathfinding.START
	mapData[job.to.Y][job.to.X] = pathfinding.STOP
	graph := pathfinding.NewGraph(&mapData)
	pathNodes := pathfinding.Astar(graph)
	zones := make([]Zone, 0, len(pathNodes))
	for _, node := range pathNodes {
		zones = append(zones, Zone{
			//lib has wrong y, x axis
			X: node.Y,
			Y: node.X,
		})
	}
	job.output = zones
	job.state = NJ_STATE_DONE
	if DEBUG_MINIMAP {
		logger.Printf("cycleID %d ScheduledPath id %d: is complete \n", CycleID, job.jobId)
	}
	return nil
}

func (receiver *Navigation) SchedulePath(from Zone, to Zone, owner PathReceiver) error {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	id := genId()
	receiver.queue = append(receiver.queue, &NavJob{
		jobId:  id,
		input:  nil,
		from:   from,
		to:     to,
		output: nil,
		state:  NJ_STATE_NEW,
		owner:  owner,
	})
	if DEBUG_MINIMAP {
		logger.Printf("cycleID %d SchedulePath id %d: %d, %d -> %d, %d \n", CycleID, id, from.X, from.Y, to.X, to.Y)
	}
	return nil
}

func NewNavigation(location *Location, collider *collider.Collider) (*Navigation, error) {
	var template pathfinding.MapData

	if location != nil {
		x, y := location.zoneX, location.zoneY
		template = *pathfinding.NewMapData(y, x)
		for r := 0; r < location.zoneY; r++ {
			for c := 0; c < location.zoneX; c++ {
				template[r][c] = pathfinding.LAND
			}
		}
	}

	return &Navigation{
		Location:        location,
		Collider:        collider,
		queue:           make([]*NavJob, 0, 10),
		mutex:           sync.Mutex{},
		mapDataTemplate: template,
		NavData:         make([][]Zone, 0, 10),
	}, nil
}
