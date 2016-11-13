package main

type pingResponse struct {
	id   int
	t    int64
	resp chan bool
}

type pingPending struct {
	id    int
	host  string
	start int64
	resp  chan bool
}

type hostResults struct {
	host        string
	lastElapsed int
	completed   int
	timeouts    int
}

type readOp struct {
	resp chan map[string]*hostResults
}
