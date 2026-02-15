package scheduler

import "time"

type Ticker struct {
	ticker *time.Ticker
}

func NewTicker(d time.Duration) *Ticker {
	return &Ticker{
		ticker: time.NewTicker(d),
	}
}

func (t *Ticker) C() <-chan time.Time {
	return t.ticker.C
}

func (t *Ticker) Stop() {
	t.ticker.Stop()
}
