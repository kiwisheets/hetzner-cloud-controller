package model

import "time"

// Service is always on port 443 so only the loadbalancer ID is used

type CertRenewJob struct {
	Model
	IntervalHours  int
	LastRun        time.Time
	NextRun        time.Time
	LoadbalancerID int
}
