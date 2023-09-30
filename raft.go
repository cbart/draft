package draft

import "context"

type Follower struct{}

func (f Follower) LocalClient() LocalClient {
	return nil
}

func (f Follower) AwaitLastLog(value string) LastLogConsensus {
	return nil
}

type Cluster []LocalClient

type LocalClient interface{}

func (c Cluster) Append(ctx context.Context, value string) error {
	return nil
}

type LastLogConsensus <-chan struct{}

func (c LastLogConsensus) Cancel() {
}
