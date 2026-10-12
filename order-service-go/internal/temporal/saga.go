package temporal

import "go.temporal.io/sdk/workflow"

// saga collects compensations and runs them in reverse order. Like the Java Saga helper with its
// default options, compensation is sequential and stops at the first compensation that fails.
type saga struct {
	compensations []func(ctx workflow.Context) error
}

func (s *saga) addCompensation(c func(ctx workflow.Context) error) {
	s.compensations = append(s.compensations, c)
}

func (s *saga) compensate(ctx workflow.Context) error {
	for i := len(s.compensations) - 1; i >= 0; i-- {
		if err := s.compensations[i](ctx); err != nil {
			return err
		}
	}
	return nil
}
