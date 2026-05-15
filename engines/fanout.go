package engines

import (
	"context"
	"sync"
	"time"

	"github.com/JoakimCarlsson/scour/query"
)

type FanOutError struct {
	Engine string
	Err    error
}

func (e *FanOutError) Error() string { return e.Engine + ": " + e.Err.Error() }
func (e *FanOutError) Unwrap() error { return e.Err }

func FanOut(
	ctx context.Context,
	q query.Query,
	engs []Engine,
	timeout time.Duration,
) ([]Result, []FanOutError) {
	all, _, errs := FanOutResponse(ctx, q, engs, timeout)
	return all, errs
}

func FanOutResponse(
	ctx context.Context,
	q query.Query,
	engs []Engine,
	timeout time.Duration,
) ([]Result, []string, []FanOutError) {
	if len(engs) == 0 {
		return nil, nil, nil
	}
	type outcome struct {
		results     []Result
		suggestions []string
		err         *FanOutError
	}
	ch := make(chan outcome, len(engs))
	var wg sync.WaitGroup
	for _, e := range engs {
		wg.Add(1)
		go func(e Engine) {
			defer wg.Done()
			eCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			resp, err := e.Search(eCtx, q)
			if err != nil {
				ch <- outcome{err: &FanOutError{Engine: e.Name(), Err: err}}
				return
			}
			ch <- outcome{results: resp.Results, suggestions: resp.Suggestions}
		}(e)
	}
	wg.Wait()
	close(ch)
	var all []Result
	var sugs []string
	var errs []FanOutError
	for o := range ch {
		if o.err != nil {
			errs = append(errs, *o.err)
			continue
		}
		all = append(all, o.results...)
		sugs = append(sugs, o.suggestions...)
	}
	return all, sugs, errs
}
