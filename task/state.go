package task


type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

var stateTransitionMap = map[State][]State{
	Pending:   []State{Scheduled},
	Scheduled: []State{Scheduled, Running, Failed},
	Running:   []State{Running, Completed, Failed},
	Completed: []State{},
	Failed:    []State{},
}

func (s State) String() []string {
	return []string{"Pending", "Scheduled", "Running", "Completed", "Failed"}
}

func Contains(states []State, state State) bool {
	for _, s := range states {
			if s == state {
				return true
			}
	}
	return false
}



func ValidStateTransition(src State, dst State) bool {
	return Contains(stateTransitionMap[src],dst)
}

