package application

import (
	"context"
	TanzError "withered/utils/error"
)

const (
	CodeEmptySystemID     = "invalid_argument.empty_system_id"
	CodeDuplicateSystemID = "invalid_state.duplicate_system_id"
)

type Scheduler[W any] struct {
	order  map[Phase][]System[W]
	phases []Phase
}

func NewScheduler[W any](phases []Phase, systems ...System[W]) (*Scheduler[W], error) {
	byPhase := make(map[Phase][]System[W])
	idToSystem := make(map[string]System[W])
	for _, sys := range systems {
		if sys.ID() == "" {
			return nil, TanzError.New("System ID cannot be empty", CodeEmptySystemID, nil)
		}
		if _, exists := idToSystem[sys.ID()]; exists {
			return nil, TanzError.New("Duplicate System ID detected", CodeDuplicateSystemID, TanzError.Fields{"system_id": sys.ID()})
		}
		idToSystem[sys.ID()] = sys
		byPhase[sys.Phase()] = append(byPhase[sys.Phase()], sys)
	}

	ordered := make(map[Phase][]System[W])
	for _, phase := range phases {
		list := byPhase[phase]
		sorted, err := topologicalSort(phase, list, idToSystem)
		if err != nil {
			return nil, err
		}
		ordered[phase] = sorted
	}

	return &Scheduler[W]{
		order:  ordered,
		phases: phases,
	}, nil
}

func (s *Scheduler[W]) RunTick(ctx context.Context, world W) {
	for _, phase := range s.phases {
		for _, sys := range s.order[phase] {
			sys.Run(ctx, world)
		}
	}
}

func topologicalSort[W any](phase Phase, systems []System[W], idToSystem map[string]System[W]) ([]System[W], error) {
	// Phase 内の System を ID でマップ化
	inPhase := make(map[string]System[W])
	for _, sys := range systems {
		inPhase[sys.ID()] = sys
	}

	dependents := make(map[string][]string) // 次に実行されるべき System ID のリスト
	inDegree := make(map[string]int)        // 依存している対象の数
	for _, sys := range systems {
		for _, depID := range sys.After() {
			depSys, exists := inPhase[depID]
			if !exists {
				return nil, TanzError.New("System dependency not found in the same phase", "invalid_state.missing_dependency", TanzError.Fields{"system_id": sys.ID(), "dependency_id": depID})
			}
			// phase 跨ぎは禁止
			if depSys.Phase() != phase {
				return nil, TanzError.New("System dependency cannot be in a different phase", "invalid_state.cross_phase_dependency", TanzError.Fields{"system_id": sys.ID(), "dependency_id": depID})
			}
			dependents[depID] = append(dependents[depID], sys.ID())
			inDegree[sys.ID()]++
		}
	}

	var queue []string
	for _, sys := range systems {
		if inDegree[sys.ID()] == 0 {
			queue = append(queue, sys.ID())
		}
	}

	sorted := make([]System[W], 0, len(systems))
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		currentSys := inPhase[currentID]
		sorted = append(sorted, currentSys)

		for _, depID := range dependents[currentID] {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				queue = append(queue, depID)
			}
		}
	}

	if len(sorted) != len(systems) {
		return nil, TanzError.New("Cyclic dependency detected among systems", "invalid_state.cyclic_dependency", nil)
	}

	return sorted, nil
}
