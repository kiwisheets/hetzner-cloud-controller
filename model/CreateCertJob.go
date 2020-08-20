package model

type CreateCertJob struct {
	Model
	LoadbalancerID    int
	ServiceListenPort int
	DomainNames       []string
}
